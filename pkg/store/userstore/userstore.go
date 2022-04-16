package userstore

import (
	"context"
	"errors"
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
	}
	return *user, nil
}
