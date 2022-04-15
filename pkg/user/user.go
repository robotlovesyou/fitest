package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
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
	FirstName       string `validate:"required"`
	LastName        string `validate:"required"`
	Nickname        string `validate:"required"`
	Password        string `validate:"required"`
	ConfirmPassword string `validate:"required"`
	Email           string `validate:"required"`
	Country         string `validate:"required"`
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
	store       UserStore
	hasher      PasswordHasher
	idGenerator IDGenerator
	validate    *validator.Validate
}

func New(store UserStore, hasher PasswordHasher, idGenerator IDGenerator, validate *validator.Validate) *Service {
	return &Service{
		store:       store,
		hasher:      hasher,
		idGenerator: idGenerator,
		validate:    validate,
	}
}

type UserStore interface {
	Create(context.Context, *userstore.User) (userstore.User, error)
}

// Interface for password hasher.
type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(hash string, plain string) bool
}

type IDGenerator func() (uuid.UUID, error)

func (service *Service) Create(ctx context.Context, newUser *NewUser) (user User, err error) {
	id, err := service.idGenerator()
	if err != nil {
		return user, err
	}

	passwordHash, err := service.hasher.Hash(newUser.Password)
	if err != nil {
		return user, err
	}

	if err = service.validate.Struct(newUser); err != nil {
		// In a real world implementation, the validation would need to return information rich enough to allow the consumer to
		// address the issue, because "computer says 'No'" is not very helpful, but it will do for here, hopefully!

		// Additionally, since this includes information which might be displayed to other users, it would likely want
		// to check for potentially offensive content in some fields
		return user, ErrInvalid
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
		if errors.Is(err, userstore.ErrAlreadyExists) {
			return user, ErrAlreadyExists
		}
		return user, fmt.Errorf("unexpected error storing user: %w", err)
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
