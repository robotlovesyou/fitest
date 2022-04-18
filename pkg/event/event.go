// package event provides a stubbed implementation of a message bus
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Result interface {
	Done(context.Context) error
}

type Bus interface {
	Send(body []byte) Result
}

type Service struct {
}

type SendResult struct {
}

func New() *Service {
	return &Service{}
}

func (SendResult) Done(ctx context.Context) error {
	select {
	case <-time.After(10 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (*Service) Send(_ []byte) Result {
	return SendResult{}
}

func SendJSON[T any](event T, bus Bus) (Result, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("cannot encode event as JSON: %w", err)
	}
	return bus.Send(body), nil
}
