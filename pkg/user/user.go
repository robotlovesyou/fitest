package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"golang.org/x/crypto/bcrypt"
)

const (
	// MaxPageLength is the maximum length of a page
	MaxPageLength = 100
	// TimeFormat is the formatting string used by the users package
	TimeFormat = time.RFC3339
	// DefaultVersion is the version for new users
	DefaultVersion = int32(1)
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
	Version      int32
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

type Service struct {
	store UserStore
	cost  int // cost for bcrypt password hashes
}

func New(store UserStore, cost int) *Service {
	return &Service{store: store, cost: cost}
}

type UserStore interface {
	Create(context.Context, *userstore.User) (userstore.User, error)
}

func (service *Service) Create(ctx context.Context, newUser *NewUser) (User, error) {
	// TODO provide a dependency to do this so that we can test failure
	id, err := uuid.NewRandom()
	if err != nil {
		panic("error handling is not implemented yet")
	}

	// TODO provide a dependency to do this so that we can test failure
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), service.cost)
	if err != nil {
		panic("error handling is not implemented yet")
	}

	usr, err := service.store.Create(ctx, &userstore.User{

		ID:           id,
		FirstName:    newUser.FirstName,
		LastName:     newUser.LastName,
		Nickname:     newUser.Nickname,
		PasswordHash: string(passwordHash),
		Email:        newUser.Email,
		Country:      newUser.Country,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Version:      DefaultVersion,
	})
	if err != nil {
		panic("error handling is not implemented yet")
	}

	return User{
		ID:           usr.ID,
		FirstName:    usr.FirstName,
		LastName:     usr.LastName,
		Nickname:     usr.Nickname,
		PasswordHash: usr.PasswordHash,
		Email:        usr.Email,
		Country:      usr.Country,
		CreatedAt:    usr.CreatedAt,
		UpdatedAt:    usr.UpdatedAt,
		Version:      usr.Version,
	}, nil
}
