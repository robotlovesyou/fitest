package userstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/utctime"
	"github.com/stretchr/testify/require"
)

func TestCanPageThroughAllUsers(t *testing.T) {
	users := make([]userstore.User, 20)
	for i := range users {
		users[i] = fakeUserRecord()
	}
	withStore(func(ctx context.Context, store *userstore.Store) {
		createMany(ctx, users, store)
		page1, err := store.FindMany(ctx, &userstore.Query{
			Page:   1,
			Length: 10,
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), page1.Page)
		require.Equal(t, int64(20), page1.Total)
		for i, itm := range page1.Items {
			compareUserRecords(t, users[i], itm)
		}

		page2, err := store.FindMany(ctx, &userstore.Query{
			Page:   2,
			Length: 10,
		})
		require.NoError(t, err)
		require.Equal(t, int64(2), page2.Page)
		require.Equal(t, int64(20), page2.Total)
		for i, itm := range page2.Items {
			compareUserRecords(t, users[i+10], itm)
		}

	})
}

func TestCanPageThroughUserFromCountry(t *testing.T) {
	users := make([]userstore.User, 20)
	for i := range users {
		if i < len(users)/2 {
			users[i] = fakeUserRecord(func(u *userstore.User) {
				u.Country = "DE"
			})
		} else {
			users[i] = fakeUserRecord(func(u *userstore.User) {
				u.Country = "NL"
			})
		}

	}
	withStore(func(ctx context.Context, store *userstore.Store) {
		createMany(ctx, users, store)
		page, err := store.FindMany(ctx, &userstore.Query{
			Page:    1,
			Length:  10,
			Country: "NL",
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), page.Page)
		require.Equal(t, int64(10), page.Total)
		for i, itm := range page.Items {
			compareUserRecords(t, users[i+10], itm)
		}
	})
}

func TestCanPageThroughUserCreatedAfter(t *testing.T) {
	users := make([]userstore.User, 20)
	for i := range users {
		if i < len(users)/2 {
			users[i] = fakeUserRecord(func(u *userstore.User) {
				u.CreatedAt = utctime.Now().Add(-24 * time.Hour)
			})
		} else {
			users[i] = fakeUserRecord(func(u *userstore.User) {
				u.CreatedAt = utctime.Now()
			})
		}

	}
	withStore(func(ctx context.Context, store *userstore.Store) {
		createMany(ctx, users, store)
		page, err := store.FindMany(ctx, &userstore.Query{
			Page:         1,
			Length:       10,
			CreatedAfter: utctime.Now().Add(-1 * time.Hour),
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), page.Page)
		require.Equal(t, int64(10), page.Total)
		for i, itm := range page.Items {
			compareUserRecords(t, users[i+10], itm)
		}
	})
}

func TestCanPageThroughUserCreatedAfterAndFromCountry(t *testing.T) {
	users := make([]userstore.User, 20)
	for i := range users {
		if i < len(users)/2 {
			users[i] = fakeUserRecord(func(u *userstore.User) {
				u.CreatedAt = utctime.Now()
				u.Country = "DE"
			})
		} else {
			users[i] = fakeUserRecord(func(u *userstore.User) {
				u.CreatedAt = utctime.Now()
				u.Country = "NL"
			})
		}

	}
	withStore(func(ctx context.Context, store *userstore.Store) {
		createMany(ctx, users, store)
		page, err := store.FindMany(ctx, &userstore.Query{
			Page:         1,
			Length:       10,
			CreatedAfter: utctime.Now().Add(-1 * time.Hour),
			Country:      "NL",
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), page.Page)
		require.Equal(t, int64(10), page.Total)
		for i, itm := range page.Items {
			compareUserRecords(t, users[i+10], itm)
		}
	})
}

func TestFindManyCanHandleEmptyResults(t *testing.T) {
	withStore(func(ctx context.Context, store *userstore.Store) {
		page, err := store.FindMany(ctx, &userstore.Query{
			Page:   1,
			Length: 10,
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), page.Page)
		require.Equal(t, int64(0), page.Total)
		require.Len(t, page.Items, 0)
	})
}
