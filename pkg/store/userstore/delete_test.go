package userstore_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/stretchr/testify/require"
)

func TestStoreCanDeleteAUserRecord(t *testing.T) {
	rec := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Create(ctx, &rec)
		require.NoError(t, err)
		err = store.DeleteOne(ctx, rec.ID)
		require.NoError(t, err)
	})
}

// Ensure that partial unique indexes are used
func TestStoreCanDeleteMultipleRecords(t *testing.T) {
	rec1 := fakeUserRecord()
	rec2 := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Create(ctx, &rec1)
		require.NoError(t, err)
		_, err = store.Create(ctx, &rec2)
		require.NoError(t, err)
		err = store.DeleteOne(ctx, rec1.ID)
		require.NoError(t, err)
		err = store.DeleteOne(ctx, rec2.ID)
		require.NoError(t, err)
	})
}

func TestStoreReturnsCorrectErrorDeletingRecordWhichDoesNotExist(t *testing.T) {
	withStore(func(ctx context.Context, store *userstore.Store) {
		err := store.DeleteOne(ctx, uuid.Must(uuid.NewRandom()))
		require.ErrorIs(t, err, userstore.ErrNotFound)
	})
}

func TestStoreCannotDeleteRecordTwice(t *testing.T) {
	rec := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Create(ctx, &rec)
		require.NoError(t, err)
		err = store.DeleteOne(ctx, rec.ID)
		require.NoError(t, err)
		err = store.DeleteOne(ctx, rec.ID)
		require.ErrorIs(t, err, userstore.ErrNotFound)
	})
}
