package userstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/stretchr/testify/require"
)

func collectEvents(ctx context.Context, store *userstore.Store, n int) []userstore.Event {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	collected := make([]userstore.Event, 0, n)
	events := store.Events(ctx, 10*time.Millisecond, 20*time.Millisecond)
	for {
		if len(collected) >= n {
			break
		}

		select {
		case <-ctx.Done():
			panic(ctx.Err())
		case e, more := <-events:
			if e.Err != nil {
				panic(e.Err)
			}
			if !more {
				break
			}
			collected = append(collected, e.Event)
			if err := store.ProcessEvent(ctx, e.Event.ID, e.Event.Version); err != nil {
				panic(err)
			}
		}
	}
	return collected
}

func TestActionsCauseEvents(t *testing.T) {
	// create a slice of cases with names, actions on the store and expected events
	cases := []struct {
		name     string
		actions  func(context.Context, *userstore.Store, *testing.T)
		expected []userstore.Action
	}{
		{
			name: "Create",
			actions: func(ctx context.Context, store *userstore.Store, t *testing.T) {
				rec := fakeUserRecord()
				_, err := store.Create(ctx, &rec)
				require.NoError(t, err)
			},
			expected: []userstore.Action{userstore.Created},
		},
		{
			name: "Create then Update",
			actions: func(ctx context.Context, store *userstore.Store, t *testing.T) {
				rec := fakeUserRecord()
				_, err := store.Create(ctx, &rec)
				require.NoError(t, err)
				_, err = store.UpdateOne(ctx, &rec)
				require.NoError(t, err)
			},
			expected: []userstore.Action{userstore.Created, userstore.Updated},
		},
		{
			name: "Create then Delete",
			actions: func(ctx context.Context, store *userstore.Store, t *testing.T) {
				rec := fakeUserRecord()
				_, err := store.Create(ctx, &rec)
				require.NoError(t, err)
				err = store.DeleteOne(ctx, rec.ID)
				require.NoError(t, err)
			},
			expected: []userstore.Action{userstore.Created, userstore.Deleted},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withStore(func(ctx context.Context, store *userstore.Store) {
				c.actions(ctx, store, t)
				events := collectEvents(ctx, store, len(c.expected))
				require.Equal(t, len(c.expected), len(events))
				for i, a := range c.expected {
					require.Equal(t, a, events[i].Action)
				}
			})
		})
	}
}

// TODO: include method and tests for marking events as done (by removing them)
