package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/stretchr/testify/require"
)

func fakeUserRef() user.Ref {
	return user.Ref{ID: uuid.Must(uuid.NewRandom()).String()}
}

func TestDeleteCallsStoreWithCorrectParameters(t *testing.T) {
	// create a fake UserRef
	// create the store stub
	// create the service
	// set up the delete stub
	// call delete on the service
	// check the response
	userRef := fakeUserRef()
	storeStub := newStubUserStore()
	withService(storeStub)(func(service *user.Service) {
		storeStub.stubDeleteOne = func(ctx context.Context, id [16]byte) error {
			idUUID := uuid.UUID(id).String()
			require.Equal(t, userRef.ID, idUUID)
			return nil
		}
		err := service.DeleteUser(context.Background(), &userRef)
		require.NoError(t, err)
	})
}
