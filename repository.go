package eventsourcing

import (
	"context"
	"errors"
	"reflect"
)

// EventIterator is the interface an event store Get needs to return
type EventIterator[T any] interface {
	Next() (Event[T], error)
	Close()
}

// EventStore interface expose the methods an event store must uphold
type EventStore[T any] interface {
	Save(events []Event[T]) error
	Get(ctx context.Context, id string, aggregateType string, afterVersion Version) (EventIterator[T], error)
}

// SnapshotStore interface expose the methods an snapshot store must uphold
type SnapshotStore interface {
	Save(s Snapshot) error
	Get(ctx context.Context, id, typ string) (Snapshot, error)
}

// Aggregate interface to use the aggregate root specific methods
type Aggregate[T any] interface {
	Root() *AggregateRoot[T]
	Transition(event Event[T])
}

type EventSubscribers[T any] interface {
	All(f func(e Event[T])) *subscription[T]
	AggregateID(f func(e Event[T]), aggregates ...Aggregate[T]) *subscription[T]
	Aggregate(f func(e Event[T]), aggregates ...Aggregate[T]) *subscription[T]
	Event(f func(e Event[T]), events ...interface{}) *subscription[T]
	Name(f func(e Event[T]), aggregate string, events ...string) *subscription[T]
}

// ErrSnapshotNotFound returns if snapshot not found
var ErrSnapshotNotFound = errors.New("snapshot not found")

// ErrAggregateNotFound returns if snapshot or event not found for aggregate
var ErrAggregateNotFound = errors.New("aggregate not found")

// Repository is the returned instance from the factory function
type Repository[T any] struct {
	eventStream *EventStream[T]
	eventStore  EventStore[T]
	snapshot    *SnapshotHandler[T]
}

// NewRepository factory function
func NewRepository[T any](eventStore EventStore[T], snapshot *SnapshotHandler[T]) *Repository[T] {
	return &Repository[T]{
		eventStore:  eventStore,
		snapshot:    snapshot,
		eventStream: NewEventStream[T](),
	}
}

// Subscribers returns an interface with all event subscribers
func (r *Repository[T]) Subscribers() EventSubscribers[T] {
	return r.eventStream
}

// Save an aggregates events
func (r *Repository[T]) Save(aggregate Aggregate[T]) error {
	root := aggregate.Root()
	// use under laying event slice to set GlobalVersion
	err := r.eventStore.Save(root.aggregateEvents)
	if err != nil {
		return err
	}
	// publish the saved events to subscribers
	r.eventStream.Publish(*root, root.Events())

	// update the internal aggregate state
	root.update()
	return nil
}

// SaveSnapshot saves the current state of the aggregate but only if it has no unsaved events
func (r *Repository[T]) SaveSnapshot(aggregate Aggregate[T]) error {
	if r.snapshot == nil {
		return errors.New("no snapshot store has been initialized")
	}
	return r.snapshot.Save(aggregate)
}

// GetWithContext fetches the aggregates event and build up the aggregate
// If there is a snapshot store try fetch a snapshot of the aggregate and fetch event after the
// version of the aggregate if any
// The event fetching can be canceled from the outside.
func (r *Repository[T]) GetWithContext(ctx context.Context, id string, aggregate Aggregate[T]) error {
	if reflect.ValueOf(aggregate).Kind() != reflect.Ptr {
		return errors.New("aggregate needs to be a pointer")
	}
	// if there is a snapshot store try fetch aggregate snapshot
	if r.snapshot != nil {
		err := r.snapshot.Get(ctx, id, aggregate)
		if err != nil && !errors.Is(err, ErrSnapshotNotFound) {
			return err
		} else if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	root := aggregate.Root()
	aggregateType := reflect.TypeOf(aggregate).Elem().Name()
	// fetch events after the current version of the aggregate that could be fetched from the snapshot store
	eventIterator, err := r.eventStore.Get(ctx, id, aggregateType, root.Version())
	if err != nil && !errors.Is(err, ErrNoEvents) {
		return err
	} else if errors.Is(err, ErrNoEvents) && root.Version() == 0 {
		// no events and no snapshot
		return ErrAggregateNotFound
	} else if ctx.Err() != nil {
		return ctx.Err()
	}
	defer eventIterator.Close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			event, err := eventIterator.Next()
			if err != nil && !errors.Is(err, ErrNoMoreEvents) {
				return err
			} else if errors.Is(err, ErrNoMoreEvents) && root.Version() == 0 {
				// no events and no snapshot (some eventstore will not return the error ErrNoEvent on Get())
				return ErrAggregateNotFound
			} else if errors.Is(err, ErrNoMoreEvents) {
				return nil
			}
			// apply the event on the aggregate
			root.BuildFromHistory(aggregate, []Event[T]{event})
		}
	}
}

// Get fetches the aggregates event and build up the aggregate
// If there is a snapshot store try fetch a snapshot of the aggregate and fetch event after the
// version of the aggregate if any
func (r *Repository[T]) Get(id string, aggregate Aggregate[T]) error {
	return r.GetWithContext(context.Background(), id, aggregate)
}
