package userstore_test

import (
	"context"
	"testing"

	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/stretchr/testify/require"
)

func TestStoreCanCreateAUserRecord(t *testing.T) {
	rec := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Create(ctx, &rec)
		require.NoError(t, err)
	})
}

func TestCannotCreateClashingRecords(t *testing.T) {
	cases := []struct {
		name  string
		userA userstore.User
		userB userstore.User
	}{
		{
			name: "Clashing Email",
			userA: fakeUserRecord(func(u *userstore.User) {
				u.Email = "abc@example.com"
			}),
			userB: fakeUserRecord(func(u *userstore.User) {
				u.Email = "abc@example.com"
			}),
		},
		{
			name: "Clashing Nickname",
			userA: fakeUserRecord(func(u *userstore.User) {
				u.Nickname = "superoriginal"
			}),
			userB: fakeUserRecord(func(u *userstore.User) {
				u.Nickname = "superoriginal"
			}),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withStore(func(ctx context.Context, store *userstore.Store) {
				_, err := store.Create(ctx, &c.userA)
				require.NoError(t, err)
				_, err = store.Create(ctx, &c.userB)
				require.ErrorIs(t, err, userstore.ErrAlreadyExists)
			})
		})
	}
}
