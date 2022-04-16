package userstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type State string

const (
	Pending    State = "Pending"
	Processing State = "Processing"
	Done       State = "Done"

	CollectionName = "users"
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
	Version      int32     `bson:"version"`
}

type Event struct {
	State     State     `bson:"state"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
	Data      User      `bson:"data"`
}

type Record struct {
	ID     uuid.UUID `bson:"_id"`
	Data   User      `bson:"data"`
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
	_, err := store.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.M{
				"data.email": 1,
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.M{
				"data.nickname": 1,
			},
			Options: options.Index().SetUnique(true),
		},
	})
	return err
}

func (store *Store) Create(ctx context.Context, user *User) (User, error) {
	rec := Record{
		ID:   user.ID,
		Data: *user,
		Events: []Event{
			{
				State:     Pending,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
				Data:      *user,
			},
		},
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
	return rec.Data, nil
}

func (store *Store) Update(ctx context.Context, update *User) (user User, err error) {

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
			"events": Event{
				State:     Pending,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
				Data:      rec,
			},
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
