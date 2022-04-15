package userstore

import (
	"errors"
	"time"
)

type State string

const (
	Pending    State = "Pending"
	Processing State = "Processing"
)

var (
	// ErrAlreadyExists is returned when the new record cannot be inserted due to a unique constraint conflict
	// In a real world implementation, this would need to carry enough information for the consumer to be able to address the issue
	ErrAlreadyExists = errors.New("a user with that email or nickname already exists")
	//ErrNotFound is returned when the requested record does not exist
	ErrNotFound = errors.New("the requested user cannot be found in the store")
)

type User struct {
	ID           [16]byte  `bson:"id"`
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
	ID     [16]byte `bson:"_id"`
	Data   User     `bson:"data"`
	Events []Event  `bson:"events"`
}
