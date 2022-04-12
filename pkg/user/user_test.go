package user_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/bxcodec/faker/v3"
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

const hashStrength = bcrypt.MinCost

func withService(store *stubUserStore, f func(service *user.Service)) {
	f(user.New(store, hashStrength))
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
func TestNewUserCallsStoreWithCorrectParameters(t *testing.T) {
	store := newStubUserStore()
	newUser := &user.NewUser{
		FirstName:       faker.FirstName(),
		LastName:        faker.LastName(),
		Nickname:        faker.Username(),
		Password:        "SuperSecretPassword",
		ConfirmPassword: "SuperSecretPassword",
		Email:           faker.Email(),
		Country:         "DE",
	}
	withService(store, func(service *user.Service) {
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
		usr, err := service.Create(context.Background(), newUser)
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
