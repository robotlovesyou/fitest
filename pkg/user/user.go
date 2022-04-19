// Package user implements the business logic of the service.
// It ensures requests are valid before submtting them to the store
// and handles publishing a stream of change events
package user

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/event"
	"github.com/robotlovesyou/fitest/pkg/log"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/telemetry"
	"github.com/robotlovesyou/fitest/pkg/utctime"
	"go.opentelemetry.io/otel"
)

const (
	// MaxPageLength is the maximum length of a page
	MaxPageLength = 100
	// TimeFormat is the formatting string used by the users package
	TimeFormat = time.RFC3339
	// DefaultVersion is the version for new users
	DefaultVersion = int64(1)
	// DefaultPage is the default page for finding users when none is provided
	DefaultPage = int64(1)
	// DefaultLength is the default page length for finding users when none is provided
	DefaultLength = int32(25)
	// MinPollInterval is the minimum time between polls for events. It should be configurable
	MinPollInterval = 10 * time.Millisecond
	// MinPollInterval is the minimum time between polls for events. It should be configurable
	MaxPollInterval = 30 * time.Millisecond
	// RetryTimeout is time an event can be left pending before retry. It should be configurable
	RetryInterval = 10 * time.Second
	// MinHealthyRatio is the minimum ratio of successful event publishes for the service to be considered healthy. It should be configurable
	MinHealthyRatio = 0.9
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

// User is the item stored by the service
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
	Version      int64
}

// Sanitized user is a User with sensitive information removed
type SanitizedUser struct {
	ID        string
	FirstName string
	LastName  string
	Nickname  string
	Email     string
	Country   string
	CreatedAt string
	UpdatedAt string
	Version   int64
}

// Update represents an update to the service
type Update struct {
	ID              string `validate:"uuid"`
	FirstName       string `validate:"required,allowed-runes"`
	LastName        string `validate:"required,allowed-runes"`
	Password        string `validate:"omitempty,min=10"`
	ConfirmPassword string `validate:"eqfield=Password"`
	Country         string `validate:"required,iso3166_1_alpha2"`
	Version         int64
}

// Event is a change message as published by the service
type Event struct {
	ID        string `json:"id"`
	Version   int64  `json:"version"`
	Action    string `json:"action"`
	CreatedAt string `json:"created_at"`
	SentAt    string `json:"sent_at"`
	Data      *SanitizedUser
}

// Ref is a reference to a single user
type Ref struct {
	ID string `validate:"uuid"`
}

// Query represents the parameters used to request a page of users
type Query struct {
	CreatedAfter string
	Country      string
	Length       int32
	Page         int64
}

// Page is a page of users
type Page struct {
	Page  int64
	Total int64
	Items []SanitizedUser
}

// Service provides the services offered by this package
type Service struct {
	store       UserStore
	hasher      PasswordHasher
	idGenerator IDGenerator
	validate    *validator.Validate
	bus         event.Bus
	eventMtx    sync.Mutex
	eventCount  int64
	successRate float64
	// In a production setting I would declare this as an interface to allow for stub implementations for testing
	// I am handling most logging at the RPC level, logging success or failure, but also need to log events, which don't exist at the RPC level
	logger *log.Logger
}

type Monitor struct {
	service *Service
}

func NewMonitor(service *Service) *Monitor {
	return &Monitor{
		service: service,
	}
}

func (m *Monitor) Name() string {
	return "Users Service"
}

func (m *Monitor) Check(context.Context) error {
	rate := m.service.CheckEventSuccessRateAndReset()
	if rate < MinHealthyRatio {
		return fmt.Errorf("Event Success is %f which is below the minimu of %f", rate, MinHealthyRatio)
	}
	return nil
}

// New creates a new service.
// It has a lot of parameters. It might be better to tidy them using an options struct
func New(store UserStore, hasher PasswordHasher, idGenerator IDGenerator, validate *validator.Validate, bus event.Bus, logger *log.Logger) *Service {
	return &Service{
		store:       store,
		hasher:      hasher,
		idGenerator: idGenerator,
		validate:    validate,
		bus:         bus,
		logger:      logger,
	}
}

// Userstore represents the fuctions which must be implemented by any storage service
type UserStore interface {
	Create(context.Context, *userstore.User) (userstore.User, error)
	UpdateOne(context.Context, *userstore.User) (userstore.User, error)
	ReadOne(context.Context, uuid.UUID) (userstore.User, error)
	DeleteOne(context.Context, uuid.UUID) error
	FindMany(context.Context, *userstore.Query) (userstore.Page, error)
	Events(context.Context, time.Duration, time.Duration, time.Duration) <-chan userstore.EventResult
	ProcessEvent(ctx context.Context, id uuid.UUID, version int64) error
}

// Interface for password hasher.
type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(hash string, plain string) bool
}

// Interface ID generation
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

// Create creates a new user if the request is valid
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
		service.logger.Errorf(ctx, err, "cannot create invalid user")
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
		CreatedAt:    utctime.Now(),
		UpdatedAt:    utctime.Now(),
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

// Update updates a user if the request is valid and references an existing user
func (service *Service) Update(ctx context.Context, update *Update) (usr User, err error) {
	if err := service.validate.Struct(update); err != nil {
		service.logger.Errorf(ctx, err, "cannot update invalid user")
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
	rec.UpdatedAt = utctime.Now()

	rec, err = service.store.UpdateOne(ctx, &rec)
	if err != nil {
		switch {
		case errors.Is(err, userstore.ErrNotFound):
			return usr, ErrNotFound
		case errors.Is(err, userstore.ErrInvalidVersion):
			return usr, ErrInvalidVersion
		default:
			return usr, fmt.Errorf("unexpected error updating user store: %w", err)
		}
	}
	return copyStoreUserToUser(&rec), nil
}

// Delete deletes a single user, if the referenced user exists
func (service *Service) Delete(ctx context.Context, ref *Ref) error {
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

// Find finds a page of users matching the given query
func (service *Service) Find(ctx context.Context, query *Query) (p Page, err error) {
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
	items := make([]SanitizedUser, 0, len(page.Items))
	for _, itm := range page.Items {
		items = append(items, *sanitizedUserFromUserstoreUser(&itm))
	}
	return Page{
		Page:  page.Page,
		Total: page.Total,
		Items: items,
	}, nil
}

func sanitizedUserFromUserstoreUser(uu *userstore.User) *SanitizedUser {
	if uu == nil {
		return nil
	}
	return &SanitizedUser{
		ID:        uu.ID.String(),
		FirstName: uu.FirstName,
		LastName:  uu.LastName,
		Nickname:  uu.Nickname,
		Email:     uu.Email,
		Country:   uu.Country,
		CreatedAt: uu.CreatedAt.Format(TimeFormat),
		UpdatedAt: uu.UpdatedAt.Format(TimeFormat),
		Version:   uu.Version,
	}
}

func eventFromUserstoreEvent(ue *userstore.Event) Event {
	return Event{
		ID:        ue.ID.String(),
		Version:   ue.Version,
		Action:    string(ue.Action),
		CreatedAt: ue.CreatedAt.Format(TimeFormat),
		SentAt:    utctime.Now().Format(TimeFormat),
		Data:      sanitizedUserFromUserstoreUser(ue.Data),
	}
}

func (service *Service) publishChange(ctx context.Context, ue userstore.Event) {
	go func() {
		ctx, cancel := context.WithTimeout(ctx, RetryInterval)
		defer cancel()

		result, err := event.SendJSON(eventFromUserstoreEvent(&ue), service.bus)
		if err != nil {
			service.logger.Errorf(ctx, err, "error sending event with id:%s and version %d", ue.ID, ue.Version)
			service.recordEventResult(false)
			return
		}
		err = result.Done(ctx)
		if err != nil {
			service.logger.Errorf(ctx, err, "did not confirm sending event with id:%s and version %d", ue.ID, ue.Version)
			service.recordEventResult(false)
			return
		}
		if err = service.store.ProcessEvent(ctx, ue.ID, ue.Version); err != nil {
			service.logger.Errorf(ctx, err, "failed to process event with id:%s and version %d", ue.ID, ue.Version)
			service.recordEventResult(false)
			return
		}
		service.logger.Infof(ctx, "send event with id: %s and version: %d", ue.ID, ue.Version)
		service.recordEventResult(true)
	}()
}

// Publish changes promots the service to start listening to the store for change events.
// and publishing to the services bus
// To stop listenting, cancel the provided context
func (service *Service) PublishChanges(ctx context.Context) {
	events := service.store.Events(ctx, MinPollInterval, MaxPollInterval, RetryInterval)
Loop:
	for {
		var result userstore.EventResult
		var more bool
		select {
		case <-ctx.Done():
			break Loop
		case result, more = <-events:
		}
		if !more {
			break Loop
		}
		//  For most tracing I am not recording the user service functions,
		// but this is the root of the calls related to event publishing
		ctx, span := otel.Tracer(telemetry.TraceName).Start(ctx, "HandlingChangeEvent")
		defer span.End()
		if result.Err != nil {
			span.RecordError(result.Err)
			service.logger.Errorf(ctx, result.Err, "error receiving event from store")
			service.recordEventResult(false)
			continue
		}
		service.publishChange(ctx, result.Event)
	}
}

func (service *Service) recordEventResult(ok bool) {
	val := float64(0.0)
	if ok {
		val = float64(1.0)
	}
	service.eventMtx.Lock()
	defer service.eventMtx.Unlock()
	service.eventCount += 1
	change := (val - service.successRate) / float64(service.eventCount)
	service.successRate += change
}

// CheckEventSuccessRateAndReset returns the proportion of events which were successfully published since the last request, and resets the counter
// It can be used by a health check
func (service *Service) CheckEventSuccessRateAndReset() float64 {
	service.eventMtx.Lock()
	defer service.eventMtx.Unlock()

	// if there have been no events, return 100%
	if service.eventCount == 0 {
		return 1.0
	}

	rate := service.successRate
	service.eventCount = 0
	service.successRate = 0.0
	return rate
}

// CheckEventCount returns the number of events recorded since the last reset
func (service *Service) CheckEventCount() int64 {
	service.eventMtx.Lock()
	defer service.eventMtx.Unlock()
	return service.eventCount
}
