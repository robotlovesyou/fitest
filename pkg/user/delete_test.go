package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/stretchr/testify/require"
)

func fakeUserRef() user.Ref {
	return user.Ref{ID: uuid.Must(uuid.NewRandom()).String()}
}

func TestDeleteCallsStoreWithCorrectParameters(t *testing.T) {
	userRef := fakeUserRef()
	storeStub := newStubUserStore()
	withService(storeStub)(func(service *user.Service) {
		storeStub.stubDeleteOne = func(ctx context.Context, id uuid.UUID) error {
			idUUID := uuid.UUID(id).String()
			require.Equal(t, userRef.ID, idUUID)
			return nil
		}
		err := service.Delete(context.Background(), &userRef)
		require.NoError(t, err)
	})
}

func TestDeleteReturnsErrorWhenRefIsInvalid(t *testing.T) {
	userRef := user.Ref{ID: "not a uuid"}
	storeStub := newStubUserStore()
	withService(storeStub)(func(service *user.Service) {
		storeStub.stubDeleteOne = func(ctx context.Context, id uuid.UUID) error {
			panic("store delete should not be called when ref is invalid")
		}
		err := service.Delete(context.Background(), &userRef)
		require.ErrorIs(t, err, user.ErrInvalid)
	})
}

func TestDeleteReturnsCorrectErrorWhenStoreDeleteFails(t *testing.T) {
	unexpected := errors.New("some unexpected error")
	cases := []struct {
		name     string
		expected error
		result   error
	}{
		{
			name:     "Not Found",
			expected: user.ErrNotFound,
			result:   userstore.ErrNotFound,
		},
		{
			name:     "Unexpected error included in chain",
			expected: unexpected,
			result:   unexpected,
		},
	}
	for _, c := range cases {
		thisCase := c
		t.Run(thisCase.name, func(t *testing.T) {
			userRef := fakeUserRef()
			storeStub := newStubUserStore()
			withService(storeStub)(func(service *user.Service) {
				storeStub.stubDeleteOne = func(ctx context.Context, id uuid.UUID) error {
					return thisCase.result
				}
				err := service.Delete(context.Background(), &userRef)
				require.ErrorIs(t, err, thisCase.expected)
			})
		})
	}
}
