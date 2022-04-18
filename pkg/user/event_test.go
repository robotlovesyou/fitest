package user_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/event"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/robotlovesyou/fitest/pkg/utctime"
	"github.com/stretchr/testify/require"
)

type sendStub func([]byte) event.Result

type eventStub struct {
	sendStub sendStub
}

func (service *eventStub) Send(body []byte) event.Result {
	return service.sendStub(body)
}

func newEventStub() *eventStub {
	return &eventStub{
		sendStub: func(body []byte) event.Result {
			panic("send stub")
		},
	}
}

type happySendResult struct{}

func (result happySendResult) Done(context.Context) error {
	return nil
}

type sadSendResult struct{}

func (result sadSendResult) Done(context.Context) error {
	return errors.New("sad send result")
}

func eventForUserRecord(uu userstore.User) userstore.Event {
	return userstore.Event{
		ID:        uu.ID,
		State:     userstore.Pending,
		Action:    userstore.Created,
		Version:   uu.Version,
		CreatedAt: utctime.Now(),
		UpdatedAt: utctime.Now(),
		Data:      &uu,
	}
}

func compareUserstoreUserAndSanitizedUser(uu *userstore.User, su *user.SanitizedUser, t *testing.T) {
	if uu == nil && su == nil {
		return
	}
	require.Equal(t, uu.ID.String(), su.ID)
	require.Equal(t, uu.FirstName, su.FirstName)
	require.Equal(t, uu.LastName, su.LastName)
	require.Equal(t, uu.Nickname, su.Nickname)
	require.Equal(t, uu.Email, su.Email)
	require.Equal(t, uu.Country, su.Country)
	require.Equal(t, uu.CreatedAt.Format(user.TimeFormat), su.CreatedAt)
	require.Equal(t, uu.UpdatedAt.Format(user.TimeFormat), su.UpdatedAt)
	require.Equal(t, uu.Version, su.Version)
}

func compareUserstoreEventAndUserEvent(use userstore.Event, ue user.Event, t *testing.T) {
	require.Equal(t, use.ID.String(), ue.ID)
	require.Equal(t, string(use.Action), ue.Action)
	require.Equal(t, use.Version, ue.Version)
	require.Equal(t, use.CreatedAt.Format(user.TimeFormat), ue.CreatedAt)
	compareUserstoreUserAndSanitizedUser(use.Data, ue.Data, t)
}

func TestReceivingAnEventFromStoreSendsADomainEvent(t *testing.T) {
	// Send `count` events from the store service.
	// Each send from the user service succeeds.
	// Compare the sent and received data

	store := newStubUserStore()
	count := 10
	recordEvents := make(map[string]userstore.Event)
	// The send stub is called inside a goroutine and accesses shared resources
	// so provide a mutex for them
	var mtx sync.Mutex
	sentEvents := make([][]byte, 0, count)
	eventStub := newEventStub()
	withService(store, useBus(eventStub))(func(service *user.Service) {
		ctx, cancel := context.WithCancel(context.Background())

		// Stub of bus.Send, which always succeeds and records the sent data
		eventStub.sendStub = func(body []byte) event.Result {
			mtx.Lock()
			defer mtx.Unlock()
			sentEvents = append(sentEvents, body)
			if len(sentEvents) >= count {
				cancel()
			}
			return happySendResult{}
		}

		// Stub of events which sends `count` events, recording each
		store.stubEvents = func(ctx context.Context, _, _, _ time.Duration) <-chan userstore.EventResult {
			out := make(chan userstore.EventResult)
			go func() {
				for n := 0; n < count; n++ {
					e := eventForUserRecord(fakeUserRecord())
					recordEvents[e.ID.String()] = e
					select {
					case out <- userstore.EventResult{Event: e}:
					case <-ctx.Done():
						return
					}
				}
			}()
			return out
		}
		store.stubProcessEvent = func(context.Context, uuid.UUID, int64) error {
			return nil
		}

		service.PublishChanges(ctx)

		// Wait until all the send goroutines complete
		for service.CheckEventCount() < int64(count) {
			time.Sleep(10 * time.Millisecond)
		}

		// Compare the events sent from the store and the events sent over the bus
		for _, sent := range sentEvents {
			var ue user.Event
			err := json.Unmarshal(sent, &ue)
			require.NoError(t, err)
			compareUserstoreEventAndUserEvent(recordEvents[ue.ID], ue, t)
		}
	})
}

func TestErrorsReceivingEventsAreRecorded(t *testing.T) {
	// Send `count` events from the user store.
	// Half the events have errors
	store := newStubUserStore()
	count := 10
	eventStub := newEventStub()

	withService(store, useBus(eventStub))(func(service *user.Service) {
		ctx, cancel := context.WithCancel(context.Background())

		// stub of bus.Send. All sends succeed
		eventStub.sendStub = func(body []byte) event.Result {
			return happySendResult{}
		}

		// stub of store.Events. Sends `count` events. Half are OK. Half have errors
		store.stubEvents = func(ctx context.Context, _, _, _ time.Duration) <-chan userstore.EventResult {
			out := make(chan userstore.EventResult)
			go func() {
				for n := 0; n < count; n++ {
					var e userstore.Event
					var err error
					if n%2 == 0 {
						e = eventForUserRecord(fakeUserRecord())
						err = nil
					} else {
						err = errors.New("some error")
					}
					select {
					case out <- userstore.EventResult{Event: e, Err: err}:
					case <-ctx.Done():
						return
					}
				}
				cancel()
			}()
			return out
		}
		store.stubProcessEvent = func(context.Context, uuid.UUID, int64) error {
			return nil
		}
		service.PublishChanges(ctx)

		// Wait until all the send goroutines complete
		for service.CheckEventCount() < int64(count) {
			time.Sleep(10 * time.Millisecond)
		}
		// math.Nextafter is suggested as the correct way to get the machine epsilon for comparing floats
		// Ensure that the success rate is 50%
		require.InDelta(t, 0.5, service.CheckEventSuccessRateAndReset(), math.Nextafter(1.0, 2.0)-1.0)
	})
}

func TestErrorsSendingEventsAreRecorded(t *testing.T) {
	// Send `count` events from the store
	// Half of the attempts to send will fail
	store := newStubUserStore()
	count := 10

	// The send event stub accesses shared resources, so provide a mutex for them
	var mtx sync.Mutex
	sent := 0

	eventStub := newEventStub()
	withService(store, useBus(eventStub))(func(service *user.Service) {
		ctx, cancel := context.WithCancel(context.Background())

		// stub of send. Half of send attempts will fail.
		eventStub.sendStub = func(body []byte) event.Result {
			mtx.Lock()
			defer mtx.Unlock()
			var result event.Result = sadSendResult{}
			if sent%2 == 0 {
				result = happySendResult{}
			}
			sent += 1
			return result
		}

		// Stub of store.Events.
		// All events succeed
		store.stubEvents = func(ctx context.Context, _, _, _ time.Duration) <-chan userstore.EventResult {
			out := make(chan userstore.EventResult)
			go func() {
				for n := 0; n < count; n++ {
					select {
					case out <- userstore.EventResult{Event: eventForUserRecord(fakeUserRecord())}:
					case <-ctx.Done():
						return
					}
				}
				cancel()
			}()
			return out
		}
		store.stubProcessEvent = func(context.Context, uuid.UUID, int64) error {
			return nil
		}
		service.PublishChanges(ctx)

		// Wait until all the send goroutines complete
		for service.CheckEventCount() < int64(count) {
			time.Sleep(10 * time.Millisecond)
		}
		// math.Nextafter is suggested as the correct way to get the machine epsilon for comparing floats
		// Check that the success rate is 50%
		require.InDelta(t, 0.5, service.CheckEventSuccessRateAndReset(), math.Nextafter(1.0, 2.0)-1.0)
	})
}
