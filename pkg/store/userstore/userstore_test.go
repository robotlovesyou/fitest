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

func createMany(ctx context.Context, items []userstore.User, store *userstore.Store) {
	for _, item := range items {
		_, err := store.Create(ctx, &item)
		if err != nil {
			panic(fmt.Errorf("cannot create many: %v", err))
		}
	}
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

func compareUserRecords(t *testing.T, a, b userstore.User) {
	require.Equal(t, a.FirstName, b.FirstName)
	require.Equal(t, a.ID, b.ID)
	require.Equal(t, a.LastName, b.LastName)
	require.Equal(t, a.Nickname, b.Nickname)
	require.Equal(t, a.PasswordHash, b.PasswordHash)
	require.Equal(t, a.Email, b.Email)
	require.Equal(t, a.Country, b.Country)
	require.True(t, b.CreatedAt.Sub(a.CreatedAt) <= time.Millisecond) // mongodb only has 1ms time resolution.
	require.True(t, b.UpdatedAt.Sub(a.UpdatedAt) <= time.Millisecond) // mongodb only has 1ms time resolution.
}

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
				u.CreatedAt = time.Now().UTC().Add(-24 * time.Hour)
			})
		} else {
			users[i] = fakeUserRecord(func(u *userstore.User) {
				u.CreatedAt = time.Now().UTC()
			})
		}

	}
	withStore(func(ctx context.Context, store *userstore.Store) {
		createMany(ctx, users, store)
		page, err := store.FindMany(ctx, &userstore.Query{
			Page:         1,
			Length:       10,
			CreatedAfter: time.Now().UTC().Add(-1 * time.Hour),
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
				u.CreatedAt = time.Now().UTC()
				u.Country = "DE"
			})
		} else {
			users[i] = fakeUserRecord(func(u *userstore.User) {
				u.CreatedAt = time.Now().UTC()
				u.Country = "NL"
			})
		}

	}
	withStore(func(ctx context.Context, store *userstore.Store) {
		createMany(ctx, users, store)
		page, err := store.FindMany(ctx, &userstore.Query{
			Page:         1,
			Length:       10,
			CreatedAfter: time.Now().UTC().Add(-1 * time.Hour),
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
