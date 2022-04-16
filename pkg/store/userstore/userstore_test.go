package userstore_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const timeout = 10 * time.Second

func testURI() (string, string) {
	uriStr := os.Getenv("MONGO_TEST_URL")
	parsed, err := url.Parse(uriStr)
	if err != nil {
		panic(fmt.Sprintf("cannot parse '%s' as a url: %v", uriStr, err))
	}

	dbName := fmt.Sprintf("db%s", uuid.Must(uuid.NewRandom()).String())

	qry := parsed.Query()
	qry.Set("authSource", "admin")
	parsed.RawQuery = qry.Encode()
	parsed.Path = dbName

	return parsed.String(), dbName
}

func withStore(f func(context.Context, *userstore.Store)) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	uri, dbName := testURI()

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(fmt.Sprintf("cannot connect to db: %v", err))
	}
	defer client.Disconnect(ctx)

	db := client.Database(dbName)
	defer db.Drop(ctx)

	store := userstore.New(db)
	if err = store.EnsureIndexes(ctx); err != nil {
		panic(fmt.Sprintf("cannot create indexes: %v", err))
	}
	f(ctx, store)
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

func TestReadOne(t *testing.T) {
	rec := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Create(ctx, &rec)
		require.NoError(t, err)
		read, err := store.ReadOne(ctx, rec.ID)
		require.NoError(t, err)
		require.Equal(t, rec.ID, read.ID)
		require.Equal(t, rec.FirstName, read.FirstName)
		require.Equal(t, rec.LastName, read.LastName)
		require.Equal(t, rec.Nickname, read.Nickname)
		require.Equal(t, rec.PasswordHash, read.PasswordHash)
		require.Equal(t, rec.Email, read.Email)
		require.Equal(t, rec.Country, read.Country)
		require.Equal(t, rec.Version, read.Version)
		require.True(t, read.CreatedAt.Sub(rec.CreatedAt) <= time.Millisecond) // mongodb only has 1ms time resolution.
		require.True(t, read.UpdatedAt.Sub(rec.UpdatedAt) <= time.Millisecond) // mongodb only has 1ms time resolution.

	})
}

func TestReadOneReturnsNotFoundWhenRecordIsMissing(t *testing.T) {
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.ReadOne(ctx, uuid.Must(uuid.NewRandom()))
		require.ErrorIs(t, err, userstore.ErrNotFound)
	})
}

func TestStoreCanUpdateAUserRecord(t *testing.T) {
	rec := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Create(ctx, &rec)
		require.NoError(t, err)
		rec.FirstName = "New"
		updated, err := store.Update(ctx, &rec)
		require.NoError(t, err)
		require.Equal(t, rec.ID, updated.ID)
		require.Equal(t, rec.FirstName, updated.FirstName)
		require.Equal(t, rec.LastName, updated.LastName)
		require.Equal(t, rec.Nickname, updated.Nickname)
		require.Equal(t, rec.PasswordHash, updated.PasswordHash)
		require.Equal(t, rec.Email, updated.Email)
		require.Equal(t, rec.Country, updated.Country)
		require.Equal(t, rec.Version+1, updated.Version)
		require.True(t, updated.CreatedAt.Sub(rec.CreatedAt) <= time.Millisecond) // mongodb only has 1ms time resolution.
		require.True(t, updated.UpdatedAt.Sub(rec.UpdatedAt) <= time.Millisecond) // mongodb only has 1ms time resolution.
	})
}

func TestUpdateFailsIfRecordDoesntExist(t *testing.T) {
	rec := fakeUserRecord()
	withStore(func(ctx context.Context, store *userstore.Store) {
		_, err := store.Update(ctx, &rec)
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
		_, err = store.Update(ctx, &rec)
		require.ErrorIs(t, err, userstore.ErrInvalidVersion)
	})
}
