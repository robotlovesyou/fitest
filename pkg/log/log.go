// package log implements a very simple structured logger by wrapping the zap logger
package log

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Key is the type for keys used by the logger context
type Key string

const (
	// The key for the request ID in the context
	RequestIDKey Key = "RequestID"

	DefaultRequestID = "None"
)

// Logger provides logging by wrapping zap sugared logger
type Logger struct {
	logger *zap.SugaredLogger
}

// Create a new Logger with the given name
func New(name string) (*Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("cannot create underlying logger: %w", err)
	}
	return &Logger{
		logger: logger.Sugar().With("name", name),
	}, nil
}

func getRequestID(ctx context.Context) string {
	raw := ctx.Value(RequestIDKey)
	if raw == nil {
		return DefaultRequestID
	}
	str, ok := raw.(string)
	if !ok {
		return DefaultRequestID
	}
	return str
}

// Infof logs an info level log which optionally includes information from the context (requestID)
func (l *Logger) Infof(ctx context.Context, format string, args ...any) {
	l.logger.Infow(fmt.Sprintf(format, args...), "request_id", getRequestID(ctx))
}

// Errorf logs an error level log which includes the provdided error and optionally includes information from the context (requestID)
func (l *Logger) Errorf(ctx context.Context, err error, format string, args ...any) {
	l.logger.Errorw(fmt.Sprintf(format, args...), "error", err.Error(), "request_id", getRequestID(ctx))
}

// WithRequestID returns a context with the provided requestId set as a value
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
