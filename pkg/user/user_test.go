package user_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bxcodec/faker/v3"
	"github.com/robotlovesyou/fitest/pkg/password"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////
//// Hand coded stub/mock for UserStore
//// I prefer hand coded stubs where appropriate because the code created by
//// mockgen makes me sad!
////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type stubCreate func(context.Context, *userstore.User) (userstore.User, error)

type stubUserStore struct {
	stubCreate stubCreate
}

func newStubUserStore() *stubUserStore {
	return &stubUserStore{
		stubCreate: func(context.Context, *userstore.User) (userstore.User, error) {
			panic("stub create")
		},
	}
}

func (store *stubUserStore) Create(ctx context.Context, rec *userstore.User) (userstore.User, error) {
	return store.stubCreate(ctx, rec)
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////
//// Test helper functions
////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type option interface {
	isoption()
}

type hasherOpt struct {
	hasher user.PasswordHasher
}

func withHasher(hasher user.PasswordHasher) hasherOpt {
	return hasherOpt{hasher: hasher}
}

func (ho hasherOpt) isoption() {}

// badHasher implements user.PasswordHasher and fails for all calls
type badHasher struct{}

func (bh badHasher) Hash(string) (string, error) {
	return "", errors.New("failed")
}

func (bh badHasher) Compare(string, string) bool {
	return false
}

func withService(store *stubUserStore, options ...option) func(func(*user.Service)) {
	hasher := user.PasswordHasher(password.NewWeak())

	for _, o := range options {
		switch opt := o.(type) {
		case hasherOpt:
			hasher = opt.hasher
		}
	}

	return func(f func(service *user.Service)) {
		f(user.New(store, hasher))
	}
}

func emptyID(id [16]byte) bool {
	var emptyID [16]byte
	return compareIDs(id, emptyID)
}

func compareIDs(a [16]byte, b [16]byte) bool {
	return bytes.Equal(a[:], b[:])
}

func checkPasswordHash(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

func fakeNewUser() user.NewUser {
	return user.NewUser{
		FirstName:       faker.FirstName(),
		LastName:        faker.LastName(),
		Nickname:        faker.Username(),
		Password:        "SuperSecretPassword",
		ConfirmPassword: "SuperSecretPassword",
		Email:           faker.Email(),
		Country:         "DE",
	}
}
func TestNewUserCallsStoreWithCorrectParameters(t *testing.T) {
	store := newStubUserStore()
	newUser := fakeNewUser()
	withService(store)(func(service *user.Service) {
		var storeUser userstore.User
		store.stubCreate = func(ctx context.Context, usr *userstore.User) (userstore.User, error) {
			storeUser = *usr
			require.False(t, emptyID(usr.ID))
			require.Equal(t, newUser.FirstName, usr.FirstName)
			require.Equal(t, newUser.LastName, usr.LastName)
			require.Equal(t, newUser.Nickname, usr.Nickname)
			require.True(t, checkPasswordHash(usr.PasswordHash, newUser.Password))
			require.Equal(t, newUser.Email, usr.Email)
			require.Equal(t, newUser.Country, usr.Country)
			require.False(t, usr.CreatedAt.IsZero())
			require.False(t, usr.UpdatedAt.IsZero())
			require.Equal(t, user.DefaultVersion, usr.Version)
			return *usr, nil
		}
		usr, err := service.Create(context.Background(), &newUser)
		require.NoError(t, err)
		require.True(t, compareIDs(usr.ID, storeUser.ID))
		require.Equal(t, newUser.FirstName, usr.FirstName)
		require.Equal(t, newUser.LastName, usr.LastName)
		require.Equal(t, newUser.Nickname, usr.Nickname)
		require.True(t, checkPasswordHash(usr.PasswordHash, newUser.Password))
		require.Equal(t, newUser.Email, usr.Email)
		require.Equal(t, newUser.Country, usr.Country)
		require.Equal(t, storeUser.CreatedAt, usr.CreatedAt)
		require.Equal(t, storeUser.UpdatedAt, usr.UpdatedAt)
		require.Equal(t, user.DefaultVersion, usr.Version)
	})
}

func TestErrorReturnedWhenHashingFails(t *testing.T) {
	store := newStubUserStore()
	newUser := fakeNewUser()
	withService(store, withHasher(badHasher{}))(func(service *user.Service) {
		_, err := service.Create(context.Background(), &newUser)
		require.Error(t, err)
	})
}
