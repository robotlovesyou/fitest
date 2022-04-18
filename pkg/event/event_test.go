package event_test

import (
	"testing"
	"time"

	"github.com/robotlovesyou/fitest/pkg/event"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

const testTimeout = 10 * time.Second

func withService(f func(context.Context, *event.Service)) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	f(ctx, event.New())
}

func TestCanSendAndWaitOnDone(t *testing.T) {
	withService(func(ctx context.Context, service *event.Service) {
		result := service.Send([]byte{1, 2, 3, 4})
		require.NoError(t, result.Done(ctx))
	})
}

type testMessage struct {
	Message string
}

func TestCanSendJSON(t *testing.T) {
	withService(func(ctx context.Context, service *event.Service) {
		result, err := event.SendJSON(testMessage{Message: "Testing"}, service)
		require.NoError(t, err)
		require.NoError(t, result.Done(ctx))
	})
}
