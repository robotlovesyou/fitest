// package event provides a stubbed implementation of a message bus
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Result of a message send.
type Result interface {
	// Blocks until message send is confirmed
	Done(context.Context) error
}

// Bus provides the ability to send messages
type Bus interface {
	Send(body []byte) Result
}

// Service implements Bus
type Service struct {
}

// SendResult implements Result
type SendResult struct {
}

func New() *Service {
	return &Service{}
}

// Done simulates waiting for send confirmation my waiting for 10ms.
// If the context is closed while waiting it will return an error
func (SendResult) Done(ctx context.Context) error {
	select {
	case <-time.After(10 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Send simply returns a SendResult
func (*Service) Send(_ []byte) Result {
	return SendResult{}
}

// SendJSON encodes event as a JSON []byte and sends it using the provided bus
func SendJSON(event any, bus Bus) (Result, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("cannot encode event as JSON: %w", err)
	}
	return bus.Send(body), nil
}
