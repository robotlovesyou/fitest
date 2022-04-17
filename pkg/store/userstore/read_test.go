package userstore_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/stretchr/testify/require"
)

func TestReadOne(t *testing.T) {
	rec := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Create(ctx, &rec)
		require.NoError(t, err)
		read, err := store.ReadOne(ctx, rec.ID)
		require.NoError(t, err)
		compareUserRecords(t, rec, read)
		require.Equal(t, rec.Version, read.Version)

	})
}

func TestReadOneReturnsNotFoundWhenRecordIsMissing(t *testing.T) {
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.ReadOne(ctx, uuid.Must(uuid.NewRandom()))
		require.ErrorIs(t, err, userstore.ErrNotFound)
	})
}
