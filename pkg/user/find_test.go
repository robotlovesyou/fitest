package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/stretchr/testify/require"
)

func fakeQuery() user.Query {
	return user.Query{
		CreatedAfter: time.Now().UTC().Add(-24 * time.Hour).Format(user.TimeFormat),
		Country:      "DE",
		Length:       10,
		Page:         int64(1),
	}
}

func fakePage(n, p int64) userstore.Page {
	items := make([]userstore.User, 0, n)
	for i := int64(0); i < n; i += 1 {
		items = append(items, fakeUserRecord())
	}
	return userstore.Page{
		Page:  p,
		Total: 10 * n,
		Items: items,
	}
}

func TestCorrectParametersPassedToStoreFind(t *testing.T) {
	// create a fake query
	// create a fake response page
	// create a stub store
	// create the service
	// set up the stub to check the incoming parameters
	// check the values in the result, including each user
	query := fakeQuery()
	page := fakePage(int64(query.Length), query.Page)
	storeStub := newStubUserStore()
	withService(storeStub)(func(service *user.Service) {
		storeStub.stubFindMany = func(ctx context.Context, q *userstore.Query) (userstore.Page, error) {
			require.Equal(t, query.CreatedAfter, q.CreatedAfter.Format(user.TimeFormat))
			require.Equal(t, query.Country, q.Country)
			require.Equal(t, query.Length, q.Length)
			require.Equal(t, query.Page, q.Page)
			return page, nil
		}
		p, err := service.FindUsers(context.Background(), &query)
		require.NoError(t, err)
		require.Equal(t, page.Page, p.Page)
		require.Equal(t, page.Total, p.Total)
		require.Len(t, p.Items, len(page.Items))
		for i, usr := range page.Items {
			require.True(t, compareIDs(usr.ID, p.Items[i].ID))
			require.Equal(t, usr.FirstName, p.Items[i].FirstName)
			require.Equal(t, usr.LastName, p.Items[i].LastName)
			require.Equal(t, usr.Nickname, p.Items[i].Nickname)
			require.Equal(t, usr.PasswordHash, p.Items[i].PasswordHash)
			require.Equal(t, usr.Email, p.Items[i].Email)
			require.Equal(t, usr.Country, p.Items[i].Country)
			require.Equal(t, usr.CreatedAt, p.Items[i].CreatedAt)
			require.Equal(t, usr.UpdatedAt, p.Items[i].UpdatedAt)
			require.Equal(t, usr.Version, p.Items[i].Version)
		}
	})

}
