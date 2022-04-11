package users

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	// MaxPageLength is the maximum length of a page
	MaxPageLength = 100
	// TimeFormat is the formatting string used by the users package
	TimeFormat = time.RFC3339
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

type Update struct {
	ID              string
	FirstName       string
	LastName        string
	Password        string
	ConfirmPassword string
	Country         string
	Version         int32
}

type Ref struct {
	ID string
}

type Query struct {
	CreatedAfter string
	Country      string
	Length       int32
	Page         int32
}

type Page struct {
	Page  int32
	Total int32
	Items []User
}
