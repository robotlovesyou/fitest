package users

import (
	"time"

	"github.com/google/uuid"
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
