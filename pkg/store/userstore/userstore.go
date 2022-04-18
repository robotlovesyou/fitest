package userstore

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/utctime"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type State string
type Action string

const (
	Pending    State = "Pending"
	Processing State = "Processing"

	Created Action = "Created"
	Updated Action = "Updated"
	Deleted Action = "Deleted"

	CollectionName = "users"

	// findTimeout is used to ensure that the goroutines created by find will complete.
	// It should probably be configurable
	findTimeout = 10 * time.Second
)

var (
	// ErrAlreadyExists is returned when the new record cannot be inserted due to a unique constraint conflict
	// In a real world implementation, this would need to carry enough information for the consumer to be able to address the issue
	ErrAlreadyExists = errors.New("a user with that email or nickname already exists")
	//ErrNotFound is returned when the requested record does not exist
	ErrNotFound = errors.New("the requested user cannot be found in the store")
	// ErrInvalidVersion is returned when a record cannot be updated because the version is out of date
	ErrInvalidVersion = errors.New("the user cannot be updated because the version is invalid")
)

type User struct {
	ID           uuid.UUID `bson:"id"`
	FirstName    string    `bson:"first_name"`
	LastName     string    `bson:"last_name"`
	Nickname     string    `bson:"nickname"`
	PasswordHash string    `bson:"password_hash"`
	Email        string    `bson:"email"`
	Country      string    `bson:"country"`
	CreatedAt    time.Time `bson:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at"`
	Version      int64     `bson:"version"`
}

type Event struct {
	ID        uuid.UUID
	State     State  `bson:"state"`
	Action    Action `bson:"action"`
	Version   int64
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
	Data      *User     `bson:"data"`
}

type EventResult struct {
	Err   error
	Event Event
}

type Record struct {
	ID     uuid.UUID `bson:"_id"`
	Data   *User     `bson:"data"`
	Events []Event   `bson:"events"`
}

type Query struct {
	CreatedAfter time.Time
	Country      string
	Length       int32
	Page         int64
}

type Page struct {
	Page  int64
	Total int64
	Items []User
}

type Store struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func New(db *mongo.Database) *Store {
	return &Store{
		db:         db,
		collection: db.Collection(CollectionName),
	}
}

func (store *Store) EnsureIndexes(ctx context.Context) error {
	// creating indexes in the foreground like this could be problematic for a production service
	_, err := store.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				bson.E{Key: "data.email", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"data": bson.M{"$type": bsontype.EmbeddedDocument}}),
		},
		{
			Keys: bson.D{
				bson.E{Key: "data.nickname", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"data": bson.M{"$type": bsontype.EmbeddedDocument}}),
		},
		{
			Keys: bson.D{
				bson.E{Key: "data.created_at", Value: 1},
				bson.E{Key: "data.country", Value: 1},
			},
		},
		{
			Keys: bson.D{
				bson.E{Key: "events.0.state", Value: 1},
				bson.E{Key: "events.0.updated_at", Value: 1},
			},
		},
	})
	return err
}

func eventFor(action Action, id uuid.UUID, version int64, user *User) Event {
	return Event{
		ID:        id,
		State:     Pending,
		Action:    action,
		Version:   version,
		CreatedAt: utctime.Now(),
		UpdatedAt: utctime.Now(),
		Data:      user,
	}
}

func (store *Store) Create(ctx context.Context, user *User) (User, error) {
	rec := Record{
		ID:     user.ID,
		Data:   user,
		Events: []Event{eventFor(Created, user.ID, user.Version, user)},
	}
	_, err := store.collection.InsertOne(ctx, &rec)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// Since there are multiple unique indexes, a real world implementation should
			// allow a consumer to differentiate between causes
			return *user, ErrAlreadyExists
		}
		return *user, fmt.Errorf("cannot store user record: %w", err)
	}
	return *user, nil
}

func (store *Store) ReadOne(ctx context.Context, id uuid.UUID) (user User, err error) {
	res := store.collection.FindOne(ctx, bson.M{
		"_id":     id,
		"data.id": id, // deleted records will not have an id value but can still have events pending
	})
	if err = res.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return user, ErrNotFound
		}
		return user, fmt.Errorf("cannot read user record: %w", err)
	}
	var rec Record
	if err = res.Decode(&rec); err != nil {
		return user, fmt.Errorf("cannot decode record: %w", err)
	}
	return *rec.Data, nil
}

func (store *Store) UpdateOne(ctx context.Context, update *User) (user User, err error) {

	rec, err := store.ReadOne(ctx, update.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return user, err
		}
		return user, fmt.Errorf("cannot read record for updating: %w", err)
	}
	if rec.Version != update.Version {
		return user, ErrInvalidVersion
	}

	rec.FirstName = update.FirstName
	rec.LastName = update.LastName
	rec.PasswordHash = update.PasswordHash
	rec.Country = update.Country
	rec.CreatedAt = update.CreatedAt
	rec.UpdatedAt = update.UpdatedAt
	rec.Version += 1

	res, err := store.collection.UpdateOne(ctx, bson.M{
		"_id":          rec.ID,
		"data.id":      rec.ID,
		"data.version": update.Version,
	}, bson.M{
		"$set": bson.M{
			"data": rec,
		},
		"$push": bson.M{
			"events": eventFor(Updated, rec.ID, rec.Version, &rec),
		},
	})
	if err != nil {
		return user, fmt.Errorf("cannot update user record: %w", err)
	}
	if res.ModifiedCount != 1 {
		// It is also possible to get here if the user was updated between the read and update calls.
		// A real world implementation may want to differentiate between those states
		return user, ErrInvalidVersion
	}
	return rec, err
}

func (store *Store) DeleteOne(ctx context.Context, id uuid.UUID) error {
	res, err := store.collection.UpdateOne(ctx, bson.M{
		"_id":     id,
		"data.id": id,
	}, bson.M{
		"$set": bson.M{
			"data": nil,
		},
		"$push": bson.M{
			"events": eventFor(Deleted, id, math.MaxInt64, nil),
		},
	})
	if err != nil {
		return fmt.Errorf("cannot delete user: %w", err)
	}
	if res.ModifiedCount != 1 {
		return ErrNotFound
	}
	return nil
}

func filterFromQuery(query *Query) bson.M {
	f := bson.M{
		"data.created_at": bson.M{"$gte": query.CreatedAfter},
	}
	if query.Country != "" {
		f["data.country"] = bson.M{"$eq": query.Country}
	}
	return f
}

func skipFromQuery(query *Query) int64 {
	skip := int64(query.Length) * (query.Page - 1)
	if skip < int64(0) {
		skip = int64(0)
	}
	return skip
}

type totalResult struct {
	count int64
	err   error
}

func (store *Store) findTotal(ctx context.Context, query *Query) <-chan totalResult {
	out := make(chan totalResult)
	go func(q Query) {
		var err error
		var count int64
		count, err = store.collection.CountDocuments(ctx, filterFromQuery(&q))
		if err != nil {
			err = fmt.Errorf("cannot count matching users: %w", err)
		}
		select {
		case <-ctx.Done():
		case out <- totalResult{count: count, err: err}:
		}
	}(*query)
	return out
}

type itemsResult struct {
	items []User
	err   error
}

func (store *Store) findItems(ctx context.Context, query *Query) <-chan itemsResult {
	out := make(chan itemsResult)
	go func(q Query) {
		items := make([]User, 0, q.Length)
		var err error
		var rec Record

		cursor, err := store.collection.Find(
			ctx,
			filterFromQuery(&q),
			options.
				Find().
				SetSort(bson.M{"data.created_at": 1}).
				SetSkip(skipFromQuery(&q)).
				SetLimit(int64(query.Length)),
		)
		if err != nil {
			err = fmt.Errorf("cannot find matching users: %w", err)
		} else {
			for {
				if err = cursor.Decode(&rec); err != nil {
					break
				}
				items = append(items, *rec.Data)
				if !cursor.Next(ctx) {
					break
				}
			}
			err = cursor.Err()
		}

		select {
		case <-ctx.Done():
		case out <- itemsResult{items: items, err: err}:
		}
	}(*query)
	return out
}

func (store *Store) FindMany(ctx context.Context, query *Query) (page Page, err error) {
	ctx, cancel := context.WithTimeout(ctx, findTimeout)
	defer cancel()

	totalChan := store.findTotal(ctx, query)
	itemsChan := store.findItems(ctx, query)
	var total totalResult
	var items itemsResult

	select {
	case <-ctx.Done():
		return page, fmt.Errorf("cannot find users total: %w", ctx.Err())
	case total = <-totalChan:
	}

	select {
	case <-ctx.Done():
		return page, fmt.Errorf("cannot find users: %w", ctx.Err())
	case items = <-itemsChan:
	}

	switch {
	case total.err != nil:
		err = total.err
	case items.err != nil:
		err = items.err
	}

	return Page{
		Page:  query.Page,
		Total: total.count,
		Items: items.items,
	}, err

}

func (store *Store) readAndUpdateNextEvent(ctx context.Context, retryTimeout time.Duration) (e Event, err error) {
	var rec Record
	res := store.collection.FindOneAndUpdate(ctx, bson.M{
		"$or": []bson.M{
			{"events.0.state": Pending},
			{
				"events.0.state":      Processing,
				"events.0.updated_at": bson.M{"$lt": utctime.Now().Add(-1 * retryTimeout)},
			},
		},
	}, bson.M{
		"$set": bson.M{
			"events.0.state":      Processing,
			"events.0.updated_at": utctime.Now(),
		},
	}, options.FindOneAndUpdate().SetSort(bson.M{"events.0.updated_at": 1}).SetReturnDocument(options.Before))
	if err = res.Err(); err != nil {
		return e, err
	}
	if err = res.Decode(&rec); err != nil {
		return e, err
	}
	return rec.Events[0], nil
}

func (store *Store) Events(ctx context.Context, minInterval, maxInterval, retryTimeout time.Duration) <-chan EventResult {
	out := make(chan EventResult)
	go func() {
		source := rand.New(rand.NewSource(utctime.Now().UnixNano()))
		for {
			var event Event
			var err error
			// read the next event in a closure so we can defer the context cancel
			func() {
				innerCtx, cancel := context.WithTimeout(ctx, findTimeout)
				defer cancel()
				event, err = store.readAndUpdateNextEvent(innerCtx, retryTimeout)
			}()
			if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
				// we can ignore this error, it just means there are no waiting events
				continue
			}
			select {
			case <-ctx.Done():
				close(out)
				return
			case out <- EventResult{Event: event, Err: err}:
			}
			waitWithJitter(ctx, minInterval, maxInterval, source)
		}
	}()
	return out
}

func waitWithJitter(ctx context.Context, minInterval, maxInterval time.Duration, source *rand.Rand) {
	min, max := int64(minInterval), int64(maxInterval)
	after := time.After(minInterval + time.Duration(source.Int63n(max-min)))
	select {
	case <-ctx.Done():
	case <-after:
	}
}

func (store *Store) ProcessEvent(ctx context.Context, id uuid.UUID, version int64) error {
	_, err := store.collection.UpdateOne(ctx, bson.M{
		"_id":                   id,
		"events.0.state":        Processing,
		"events.0.data.version": version,
	}, bson.M{
		"$pop": bson.M{"events": -1},
	})
	if err != nil {
		err = fmt.Errorf("cannot complete event: %w", err)
	}
	return err
}
