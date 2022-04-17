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
