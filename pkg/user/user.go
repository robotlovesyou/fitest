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
	// DefaultPage is the default page for finding users when none is provided
	DefaultPage = int64(1)
	// DefaultLength is the default page length for finding users when none is provided
	DefaultLength = int32(25)
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
	FirstName       string `validate:"required,allowed-runes"`
	LastName        string `validate:"required,allowed-runes"`
	Nickname        string `validate:"required,allowed-runes"`
	Password        string `validate:"min=10"`
	ConfirmPassword string `validate:"required,eqfield=Password"`
	Email           string `validate:"required,email"`
	Country         string `validate:"required,iso3166_1_alpha2"`
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
	ID              string `validate:"uuid"`
	FirstName       string `validate:"required,allowed-runes"`
	LastName        string `validate:"required,allowed-runes"`
	Password        string `validate:"omitempty,min=10"`
	ConfirmPassword string `validate:"eqfield=Password"`
	Country         string `validate:"required,iso3166_1_alpha2"`
	Version         int32
}

type Ref struct {
	ID string `validate:"uuid"`
}

type Query struct {
	CreatedAfter string
	Country      string
	Length       int32
	Page         int64
}

type Page struct {
	Page  int64
	Total int64
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
	Update(context.Context, *userstore.User) (userstore.User, error)
	ReadOne(context.Context, uuid.UUID) (userstore.User, error)
	DeleteOne(context.Context, uuid.UUID) error
	FindMany(context.Context, *userstore.Query) (userstore.Page, error)
}

// Interface for password hasher.
type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(hash string, plain string) bool
}

type IDGenerator func() (uuid.UUID, error)

func copyStoreUserToUser(usr *userstore.User) User {
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
	}
}

func (service *Service) Create(ctx context.Context, newUser *NewUser) (user User, err error) {
	id, err := service.idGenerator()
	if err != nil {
		return user, fmt.Errorf("cannot generate uuid: %w", err)
	}

	passwordHash, err := service.hasher.Hash(newUser.Password)
	if err != nil {
		return user, fmt.Errorf("cannot hash password: %w", err)
	}

	if err = service.validate.Struct(newUser); err != nil {
		// In a real world implementation, the validation would need to return information rich enough to allow the consumer to
		// address the issue, because "computer says 'No'" is not very helpful, but it will do for here, hopefully!

		// Additionally, since this includes information which might be displayed to other users, it would likely want
		// to check for potentially offensive content in some fields
		return user, ErrInvalid
	}

	rec, err := service.store.Create(ctx, &userstore.User{
		ID:           id,
		FirstName:    newUser.FirstName,
		LastName:     newUser.LastName,
		Nickname:     newUser.Nickname,
		PasswordHash: string(passwordHash),
		Email:        newUser.Email,
		Country:      newUser.Country,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		Version:      DefaultVersion,
	})
	if err != nil {
		if errors.Is(err, userstore.ErrAlreadyExists) {
			return user, ErrAlreadyExists
		}
		return user, fmt.Errorf("unexpected error storing user: %w", err)
	}

	return copyStoreUserToUser(&rec), nil
}

func (service *Service) updateHashIfSet(update *Update, rec *userstore.User) (err error) {
	if len(update.Password) == 0 {
		return nil
	}
	rec.PasswordHash, err = service.hasher.Hash(update.Password)
	if err != nil {
		return fmt.Errorf("cannot update password hash: %w", err)
	}
	return
}

func (service *Service) Update(ctx context.Context, update *Update) (usr User, err error) {
	if err := service.validate.Struct(update); err != nil {
		// In a real world implementation, the validation would need to return information rich enough to allow the consumer to
		// address the issue, because "computer says 'No'" is not very helpful, but it will do for here, hopefully!
		return usr, ErrInvalid
	}

	id := uuid.MustParse(update.ID) // ok to call function which can panic because id has already been validated as a uuid

	rec, err := service.store.ReadOne(ctx, id)
	if err != nil {
		if errors.Is(err, userstore.ErrNotFound) {
			return usr, ErrNotFound
		}
	}
	if update.Version != rec.Version {
		return usr, ErrInvalidVersion
	}

	if err = service.updateHashIfSet(update, &rec); err != nil {
		return usr, err
	}

	rec.FirstName = update.FirstName
	rec.LastName = update.LastName
	rec.Country = update.Country
	rec.UpdatedAt = time.Now().UTC()
	rec.Version += 1

	rec, err = service.store.Update(ctx, &rec)
	if err != nil {
		if errors.Is(err, userstore.ErrNotFound) {
			// For the sake of simplicity I am assuming that if the userstore returns not found then the version
			// is no longer valid. A real world implementation should probably be able to distinguish between that
			// and the record having been deleted
			return usr, ErrInvalidVersion
		}
		return usr, fmt.Errorf("unexpected error updating user store: %w", err)
	}
	return copyStoreUserToUser(&rec), nil
}

func (service *Service) DeleteUser(ctx context.Context, ref *Ref) error {
	if err := service.validate.Struct(ref); err != nil {
		return ErrInvalid
	}

	id := uuid.MustParse(ref.ID) // TODO: Ensure this is validated before call
	if err := service.store.DeleteOne(ctx, id); err != nil {
		if errors.Is(err, userstore.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("cannot delete user: %w", err)
	}

	return nil
}

func (service *Service) FindUsers(ctx context.Context, query *Query) (p Page, err error) {
	ca, err := time.Parse(TimeFormat, query.CreatedAfter)
	if err != nil {
		ca = time.Time{} // pass zero time as the default, because everything is created afterward
		// This approach could be problematic if users are submitting badly formatted dates because
		// it hides the error. One solution might be to return the query as it was understoof by the service
	}
	if query.Page == 0 {
		query.Page = DefaultPage
	}
	if query.Length == 0 {
		query.Length = DefaultLength
	}
	page, err := service.store.FindMany(ctx, &userstore.Query{
		CreatedAfter: ca,
		Country:      query.Country,
		Length:       query.Length,
		Page:         query.Page,
	})
	if err != nil {
		return p, fmt.Errorf("cannot find users in store: %w", err)
	}
	items := make([]User, 0, len(page.Items))
	for _, itm := range page.Items {
		items = append(items, copyStoreUserToUser(&itm))
	}
	return Page{
		Page:  page.Page,
		Total: page.Total,
		Items: items,
	}, nil
}
