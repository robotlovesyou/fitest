package user_test

import (
	"context"
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
