package log

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type Key string

const (
	RequestIDKey Key = "RequestID"

	DefaultRequestID = "None"
)

type Logger struct {
	logger *zap.SugaredLogger
}

func New(name string) (*Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("cannot create underlying logger: %w", err)
	}
	return &Logger{
		logger: logger.Sugar().With("name", name),
	}, nil
}

func GetRequestID(ctx context.Context) string {
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

func (l *Logger) Infof(ctx context.Context, format string, args ...any) {
	l.logger.Infow(fmt.Sprintf(format, args...), "request_id", GetRequestID(ctx))
}

func (l *Logger) Errorf(ctx context.Context, err error, format string, args ...any) {
	l.logger.Errorw(fmt.Sprintf(format, args...), "error", err.Error(), "request_id", GetRequestID(ctx))
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
