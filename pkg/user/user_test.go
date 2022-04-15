package user_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bxcodec/faker/v3"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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

func useHasher(hasher user.PasswordHasher) hasherOpt {
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

type idGenOpt struct {
	idGenerator user.IDGenerator
}

func useIDGenerator(idGenerator user.IDGenerator) idGenOpt {
	return idGenOpt{idGenerator: idGenerator}
}

func (igo idGenOpt) isoption() {}

func withService(store *stubUserStore, options ...option) func(func(*user.Service)) {
	hasher := user.PasswordHasher(password.NewWeak())
	idGenerator := uuid.NewRandom

	for _, o := range options {
		switch opt := o.(type) {
		case hasherOpt:
			hasher = opt.hasher
		case idGenOpt:
			idGenerator = opt.idGenerator
		}
	}

	return func(f func(service *user.Service)) {
		f(user.New(store, hasher, idGenerator, validator.New()))
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

func fakeNewUser(muts ...func(*user.NewUser)) user.NewUser {
	nu := user.NewUser{
		FirstName:       faker.FirstName(),
		LastName:        faker.LastName(),
		Nickname:        faker.Username(),
		Password:        "SuperSecretPassword",
		ConfirmPassword: "SuperSecretPassword",
		Email:           faker.Email(),
		Country:         "DE",
	}

	for _, m := range muts {
		m(&nu)
	}
	return nu
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
	withService(store, useHasher(badHasher{}))(func(service *user.Service) {
		_, err := service.Create(context.Background(), &newUser)
		require.Error(t, err)
	})
}

func TestErrorReturnedWhenIDGenerationFails(t *testing.T) {
	store := newStubUserStore()
	newUser := fakeNewUser()
	badIDGenerator := func() (uuid.UUID, error) { return uuid.Nil, errors.New("failed") }
	withService(store, useIDGenerator(badIDGenerator))(func(service *user.Service) {
		_, err := service.Create(context.Background(), &newUser)
		require.Error(t, err)
	})
}

func TestCorrectErrorIsReturnedWhenStoreReturnsErrAlreadyExists(t *testing.T) {
	unexpected := errors.New("unexpected")
	cases := []struct {
		name     string
		expected error
		result   error
	}{
		{
			name:     "Already Exists",
			expected: user.ErrAlreadyExists,
			result:   userstore.ErrAlreadyExists,
		},
		{
			name:     "Unepected Error included in chain",
			expected: unexpected,
			result:   unexpected,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := newStubUserStore()
			newUser := fakeNewUser()
			withService(store)(func(service *user.Service) {
				store.stubCreate = func(context.Context, *userstore.User) (usr userstore.User, err error) {
					return usr, c.result
				}
				_, err := service.Create(context.Background(), &newUser)
				require.ErrorIs(t, err, c.expected)
			})
		})
	}

}

func TestCorrectErrorIsReturnedForInvalidNewUser(t *testing.T) {
	// Test for each required field missing
	// Test for illegal values in each field
	// Test for invalid email
	// Test for password too short
	// Test for password and confirmation do not match

	// In a real world implementation, the validation would need to return information rich enough to allow the consumer to
	// address the issue, because "computer says 'No'" is not very helpful, but it will do for here, hopefully!
	cases := []struct {
		name    string
		newUser user.NewUser
	}{
		{
			name: "No first name",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.FirstName = ""
			}),
		},
		{
			name: "No last name",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.LastName = ""
			}),
		},
		{
			name: "No Nickname",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.Nickname = ""
			}),
		},
		{
			name: "No Password",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.Password = ""
			}),
		},
		{
			name: "No Password Confirmation",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.ConfirmPassword = ""
			}),
		},
		{
			name: "No Email",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.Email = ""
			}),
		},
		{
			name: "No Country",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.Country = ""
			}),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := newStubUserStore()
			withService(store)(func(service *user.Service) {
				store.stubCreate = func(context.Context, *userstore.User) (userstore.User, error) {
					panic("should not be calling store with invalid new user")
				}
				_, err := service.Create(context.Background(), &c.newUser)
				require.ErrorIs(t, err, user.ErrInvalid)
			})
		})
	}
}
