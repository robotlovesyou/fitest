package log_test

import (
	"context"
	"errors"
	"testing"

	"github.com/robotlovesyou/fitest/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestCanCallInfoWithNoTraceID(t *testing.T) {
	l, err := log.New("test")
	require.NoError(t, err)
	l.Infof(context.Background(), "test message %d", 123)
}

func TestCanCallInfoWithTestID(t *testing.T) {
	l, err := log.New("test")
	require.NoError(t, err)
	l.Infof(log.WithRequestID(context.Background(), "test_request_id"), "test message %d", 123)
}

func TestCanCallErrorWithNoTraceID(t *testing.T) {
	l, err := log.New("test")
	require.NoError(t, err)
	l.Errorf(context.Background(), errors.New("test error"), "test message %d", 123)
}

func TestCanCallErrorWithTestID(t *testing.T) {
	l, err := log.New("test")
	require.NoError(t, err)
	l.Errorf(log.WithRequestID(context.Background(), "test_request_id"), errors.New("test error"), "test message %d", 123)
}
