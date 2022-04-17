package userstore_test

import (
	"context"
	"testing"

	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/stretchr/testify/require"
)

func TestStoreCanUpdateAUserRecord(t *testing.T) {
	rec := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Create(ctx, &rec)
		require.NoError(t, err)
		rec.FirstName = "New"
		updated, err := store.UpdateOne(ctx, &rec)
		require.NoError(t, err)
		compareUserRecords(t, rec, updated)
		require.Equal(t, rec.Version+1, updated.Version)
	})
}

func TestUpdateFailsIfRecordDoesntExist(t *testing.T) {
	rec := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.UpdateOne(ctx, &rec)
		require.ErrorIs(t, err, userstore.ErrNotFound)
	})
}

func TestUpdateFailsIfUpdateVersionIsStale(t *testing.T) {
	rec := fakeUserRecord()
	rec.Version = 2
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Create(ctx, &rec)
		require.NoError(t, err)
		rec.FirstName = "New"
		rec.Version = 1
		_, err = store.UpdateOne(ctx, &rec)
		require.ErrorIs(t, err, userstore.ErrInvalidVersion)
	})
}
