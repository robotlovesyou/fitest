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
)

type NewUser struct {
	FirstName       string `faker:"first_name"`
	LastName        string `faker:"last_name"`
	Nickname        string `faker:"username"`
	Password        string `faker:"password"`
	ConfirmPassword string
	Email           string `faker:"email"`
	Country         string `faker:"country"`
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
