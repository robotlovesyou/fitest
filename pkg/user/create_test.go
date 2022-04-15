package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bxcodec/faker/v3"
	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/stretchr/testify/require"
)

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
	// In a real world implementation, the validation would need to return information rich enough to allow the consumer to
	// address the issue, because "computer says 'No'" is not very helpful, but it will do for here, hopefully!
	cases := []struct {
		name    string
		newUser user.NewUser
	}{
		// Tests for missing fields
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
		// Tests for invalid fields
		{
			name: "Bad First Name",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.FirstName = bobbyTables
			}),
		},
		{
			name: "Bad last name",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.LastName = bobbyTables
			}),
		},
		{
			name: "Bad Nickname",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.Nickname = bobbyTables
			}),
		},
		{
			name: "Bad Email",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.Email = "not an email address"
			}),
		},
		{
			name: "Bad Country",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.Country = "123"
			}),
		},
		// Bad Password Tests
		// Password Policies are often more complex than implemented here
		{
			name: "Bad Password Conformation",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.Password = "supersecret"
				nu.ConfirmPassword = "notsupersecret"
			}),
		},
		{
			name: "Short Password",
			newUser: fakeNewUser(func(nu *user.NewUser) {
				nu.Password = "short"
				nu.ConfirmPassword = "short"
			}),
		},
	}
	for _, c := range cases {
		thisCase := c
		t.Run(thisCase.name, func(t *testing.T) {
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
