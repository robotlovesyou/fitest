package users

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrAlreadyExists is returned when the users email address or nickname are not unique.
	// In a real world implementation further detail would be required to allow the client to rectify the error
	ErrAlreadyExists = errors.New("user with that email or nickname already exists")
	// ErrInvalid is returned when the validation of a new or updated user fails
	// In a real world implementation further detail would be required to allow the client to rectify the error
	ErrInvalid = errors.New("user is invalid")
	// ErrInvalidVersion is returned when the version returned with the update is incorrect, which would indicate that the
	// data is stale
	ErrInvalidVersion = errors.New("version is invalid")
	// ErrNotFound is returned when the user matching a request does not exist
	ErrNotFound = errors.New("user not found")
)

type NewUser struct {
	FirstName       string
	LastName        string
	Nickname        string
	Password        string
	ConfirmPassword string
	Email           string
	Country         string
}

type User struct {
	ID           uuid.UUID
	FirstName    string
	LastName     string
	Nickname     string
	PasswordHash string
	Email        string
	Country      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserUpdate struct {
	ID              string
	FirstName       string
	LastName        string
	Password        string
	ConfirmPassword string
	Country         string
	Version         int32
}
