package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/stretchr/testify/require"
)

func fakeUserUpdate(muts ...func(u *user.Update)) user.Update {
	password := faker.Password()

	upd := user.Update{
		ID:              uuid.Must(uuid.NewRandom()).String(),
		FirstName:       faker.FirstName(),
		LastName:        faker.LastName(),
		Password:        password,
		ConfirmPassword: password,
		Country:         "NL",
		Version:         user.DefaultVersion,
	}

	for _, m := range muts {
		m(&upd)
	}
	return upd
}

func fakeUserRecord(muts ...func(r *userstore.User)) userstore.User {
	r := userstore.User{
		ID:           uuid.Must(uuid.NewRandom()),
		FirstName:    faker.FirstName(),
		LastName:     faker.LastName(),
		Nickname:     faker.Username(),
		PasswordHash: "supersecrethash",
		Email:        faker.Email(),
		Country:      "DE",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Version:      user.DefaultVersion,
	}

	for _, m := range muts {
		m(&r)
	}
	return r
}

func TestUpdateUserCallsStoreWithCorrectParameters(t *testing.T) {
	store := newStubUserStore()
	update := fakeUserUpdate()
	rec := fakeUserRecord(func(r *userstore.User) {
		r.ID = uuid.MustParse(update.ID)
	})

	withService(store)(func(service *user.Service) {
		var storeUser userstore.User
		store.stubReadOne = func(context.Context, [16]byte) (userstore.User, error) {
			return rec, nil
		}
		store.stubUpdate = func(ctx context.Context, usr *userstore.User) (userstore.User, error) {
			storeUser = *usr
			require.False(t, emptyID(usr.ID))
			require.Equal(t, update.FirstName, usr.FirstName)
			require.Equal(t, update.LastName, usr.LastName)
			require.Equal(t, rec.Nickname, usr.Nickname)
			require.True(t, checkPasswordHash(usr.PasswordHash, update.Password))
			require.Equal(t, rec.Email, usr.Email)
			require.Equal(t, update.Country, usr.Country)
			require.False(t, usr.CreatedAt.IsZero())
			require.False(t, usr.UpdatedAt.IsZero())
			require.True(t, usr.UpdatedAt.After(rec.UpdatedAt))
			require.True(t, usr.Version > rec.Version)
			return *usr, nil
		}
		usr, err := service.Update(context.Background(), &update)
		require.NoError(t, err)
		require.True(t, compareIDs(usr.ID, storeUser.ID))
		require.Equal(t, update.FirstName, usr.FirstName)
		require.Equal(t, update.LastName, usr.LastName)
		require.Equal(t, rec.Nickname, usr.Nickname)
		require.True(t, checkPasswordHash(usr.PasswordHash, update.Password))
		require.Equal(t, rec.Email, usr.Email)
		require.Equal(t, update.Country, usr.Country)
		require.Equal(t, rec.CreatedAt, usr.CreatedAt)
		require.True(t, rec.UpdatedAt.Before(usr.UpdatedAt))
		require.Equal(t, update.Version+1, usr.Version)
	})
}

func TestForErrorWhenUpdateContainsInvalidValues(t *testing.T) {
	// In a real world implementation, the validation would need to return information rich enough to allow the consumer to
	// address the issue, because "computer says 'No'" is not very helpful, but it will do for here, hopefully!
	cases := []struct {
		name   string
		update user.Update
	}{
		{
			name: "Bad ID",
			update: fakeUserUpdate(func(u *user.Update) {
				u.ID = "not a uuid"
			}),
		},
		{
			name: "No First Name",
			update: fakeUserUpdate(func(u *user.Update) {
				u.FirstName = ""
			}),
		},
		{
			name: "No Last Name",
			update: fakeUserUpdate(func(u *user.Update) {
				u.LastName = ""
			}),
		},
		{
			name: "No Country",
			update: fakeUserUpdate(func(u *user.Update) {
				u.Country = ""
			}),
		},
		{
			name: "Bad First Name",
			update: fakeUserUpdate(func(u *user.Update) {
				u.FirstName = bobbyTables
			}),
		},
		{
			name: "Bad Last Name",
			update: fakeUserUpdate(func(u *user.Update) {
				u.LastName = bobbyTables
			}),
		},
		{
			name: "Bad Country",
			update: fakeUserUpdate(func(u *user.Update) {
				u.Country = "123"
			}),
		},
		{
			name: "Passwords Don't Match",
			update: fakeUserUpdate(func(u *user.Update) {
				u.ConfirmPassword = "not the same as password"
			}),
		},
		{
			name: "Password Too Short",
			update: fakeUserUpdate(func(u *user.Update) {
				u.Password = "short"
				u.ConfirmPassword = "short"
			}),
		},
	}
	for _, c := range cases {
		thisCase := c
		t.Run(thisCase.name, func(t *testing.T) {
			store := newStubUserStore()
			withService(store)(func(service *user.Service) {
				store.stubReadOne = func(context.Context, [16]byte) (userstore.User, error) {
					panic("should not be calling read one when update is invalid")
				}
				store.stubUpdate = func(context.Context, *userstore.User) (userstore.User, error) {
					panic("should not be calling update when update is invalid")
				}
				_, err := service.Update(context.Background(), &c.update)
				require.ErrorIs(t, err, user.ErrInvalid)
			})
		})
	}
}

func TestPasswordIsNotReHashedWhenItIsNotSetInTheUpdate(t *testing.T) {
	store := newStubUserStore()
	update := fakeUserUpdate(func(u *user.Update) {
		u.Password = ""
		u.ConfirmPassword = ""
	})
	rec := fakeUserRecord(func(r *userstore.User) {
		r.ID = uuid.MustParse(update.ID)
	})

	withService(store)(func(service *user.Service) {
		store.stubReadOne = func(context.Context, [16]byte) (userstore.User, error) {
			return rec, nil
		}
		store.stubUpdate = func(ctx context.Context, usr *userstore.User) (userstore.User, error) {
			return *usr, nil
		}
		usr, err := service.Update(context.Background(), &update)
		require.NoError(t, err)
		require.Equal(t, rec.PasswordHash, usr.PasswordHash)
	})
}

func TestForErrorUpdatingUserWhenRecordNotFound(t *testing.T) {
	store := newStubUserStore()
	update := fakeUserUpdate()

	withService(store)(func(service *user.Service) {
		store.stubReadOne = func(context.Context, [16]byte) (rec userstore.User, err error) {
			return rec, userstore.ErrNotFound
		}
		store.stubUpdate = func(ctx context.Context, usr *userstore.User) (userstore.User, error) {
			panic("should not be calling update when the record is not found")
		}
		_, err := service.Update(context.Background(), &update)
		require.ErrorIs(t, err, user.ErrNotFound)
	})
}

func TestForErrorUpdatingUserWhenPasswordCannotBeHashed(t *testing.T) {
	store := newStubUserStore()
	update := fakeUserUpdate()
	rec := fakeUserRecord(func(r *userstore.User) {
		r.ID = uuid.MustParse(update.ID)
	})

	withService(store, useHasher(badHasher{}))(func(service *user.Service) {
		store.stubReadOne = func(context.Context, [16]byte) (userstore.User, error) {
			return rec, nil
		}
		store.stubUpdate = func(ctx context.Context, usr *userstore.User) (userstore.User, error) {
			return *usr, nil
		}
		_, err := service.Update(context.Background(), &update)
		require.Error(t, err)
	})
}

func TestForErrorUpdatingUserWhenVersionIsStale(t *testing.T) {
	store := newStubUserStore()
	update := fakeUserUpdate()
	rec := fakeUserRecord(func(r *userstore.User) {
		r.ID = uuid.MustParse(update.ID)
		r.Version = update.Version + 1
	})

	withService(store)(func(service *user.Service) {
		store.stubReadOne = func(context.Context, [16]byte) (userstore.User, error) {
			return rec, nil
		}
		store.stubUpdate = func(ctx context.Context, usr *userstore.User) (userstore.User, error) {
			panic("should not be calling store update when version is stale")
		}
		_, err := service.Update(context.Background(), &update)
		require.ErrorIs(t, err, user.ErrInvalidVersion)
	})
}

func TestForErrorUpdatingUserWhenStoreUpdateFails(t *testing.T) {
	unexpected := errors.New("unexpected")
	cases := []struct {
		name     string
		expected error
		result   error
	}{
		{
			name:     "Bad ID",
			expected: user.ErrInvalidVersion,
			result:   userstore.ErrNotFound,
		},
		{
			name:     "Unexpected Error From Store Is Included In Chain",
			expected: unexpected,
			result:   unexpected,
		},
	}
	for _, c := range cases {
		thisCase := c
		t.Run(thisCase.name, func(t *testing.T) {
			store := newStubUserStore()
			update := fakeUserUpdate()
			rec := fakeUserRecord(func(r *userstore.User) {
				r.ID = uuid.MustParse(update.ID)
			})
			withService(store)(func(service *user.Service) {
				store.stubReadOne = func(context.Context, [16]byte) (userstore.User, error) {
					return rec, nil
				}
				store.stubUpdate = func(context.Context, *userstore.User) (rec userstore.User, err error) {
					return rec, thisCase.result
				}
				_, err := service.Update(context.Background(), &update)
				require.ErrorIs(t, err, thisCase.expected)
			})
		})
	}
}
