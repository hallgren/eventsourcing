package aggregate

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/internal"
)

type RegisterFunc = func(events ...interface{})

// register holding functions that is triggered when events for an aggregate is saved.
var saveHookMap = make(map[string]func(events []eventsourcing.Event))

// Aggregate interface to use the aggregate root specific methods
type aggregate interface {
	root() *Root
	Transition(event eventsourcing.Event)
	Register(RegisterFunc)
}

// Load returns the aggregate based on its events
func Load(ctx context.Context, es core.EventStore, id string, a aggregate) error {
	if reflect.ValueOf(a).Kind() != reflect.Ptr {
		return eventsourcing.ErrAggregateNeedsToBeAPointer
	}

	root := a.root()

	iterator, err := getEvents(ctx, es, id, aggregateType(a), root.Version())
	if err != nil {
		return err
	}
	defer iterator.Close()
	for iterator.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			event, err := iterator.Value()
			if err != nil {
				return err
			}
			buildFromHistory(a, []eventsourcing.Event{event})
		}
	}
	if root.Version() == 0 {
		return eventsourcing.ErrAggregateNotFound
	}
	return nil
}

// LoadFromSnapshot fetch the aggregate by first get its snapshot and later append events after the snapshot was stored
// This can speed up the load time of aggregates with many events
func LoadFromSnapshot(ctx context.Context, es core.EventStore, ss core.SnapshotStore, id string, as aggregateSnapshot) error {
	err := LoadSnapshot(ctx, ss, id, as)
	if err != nil {
		return err
	}
	return Load(ctx, es, id, as)
}

// Save stores the aggregate events in the supplied event store
func Save(es core.EventStore, a aggregate) error {
	root := a.root()

	// return as quick as possible when no events to process
	if len(root.events) == 0 {
		return nil
	}

	if !internal.GlobalRegister.AggregateRegistered(a) {
		return fmt.Errorf("%s %w", aggregateType(a), eventsourcing.ErrAggregateNotRegistered)
	}

	globalVersion, err := saveEvents(es, root.Events())
	if err != nil {
		return err
	}
	// set the global version
	root.globalVersion = globalVersion

	// set the  internal version
	lastEvent := root.events[len(root.events)-1]
	root.version = lastEvent.Version()

	// run save hook function for the aggregate type if its been set
	if f, ok := saveHookMap[aggregateType(a)]; ok {
		f(root.events)
	}

	// clear the events
	root.events = []eventsourcing.Event{}
	return nil
}

// Register registers the aggregate and its events
func Register(a aggregate) {
	internal.GlobalRegister.Register(a)
}

// SetSaveHook enables the application to react in realtime for saved events from specific aggregates.
// Note that the function is ran in sync with the Save method and should return as fast as possible.
// It return error if an aggregate is not registered via the aggregate.Register function.
func SetSaveHook(f func(events []eventsourcing.Event), aggregates ...aggregate) error {
	for _, a := range aggregates {
		if !internal.GlobalRegister.AggregateRegistered(a) {
			return fmt.Errorf("%s %w when calling the SetSaveHook", aggregateType(a), eventsourcing.ErrAggregateNotRegistered)
		}
		saveHookMap[aggregateType(a)] = f
	}
	return nil
}

// Save events to the event store
func saveEvents(eventStore core.EventStore, events []eventsourcing.Event) (eventsourcing.Version, error) {
	var esEvents = make([]core.Event, 0, len(events))

	for _, event := range events {
		data, err := internal.EventEncoder.Serialize(event.Data())
		if err != nil {
			return 0, err
		}
		metadata, err := internal.EventEncoder.Serialize(event.Metadata())
		if err != nil {
			return 0, err
		}

		esEvent := core.Event{
			AggregateID:   event.AggregateID(),
			Version:       core.Version(event.Version()),
			AggregateType: event.AggregateType(),
			Timestamp:     event.Timestamp(),
			Data:          data,
			Metadata:      metadata,
			Reason:        event.Reason(),
		}
		_, ok := internal.GlobalRegister.EventRegistered(esEvent)
		if !ok {
			return 0, fmt.Errorf("%s %w", esEvent.Reason, eventsourcing.ErrEventNotRegistered)
		}
		esEvents = append(esEvents, esEvent)
	}

	err := eventStore.Save(esEvents)
	if err != nil {
		if errors.Is(err, core.ErrConcurrency) {
			return 0, eventsourcing.ErrConcurrency
		}
		return 0, fmt.Errorf("error from event store: %w", err)
	}

	return eventsourcing.Version(esEvents[len(esEvents)-1].GlobalVersion), nil
}

// getEvents return event iterator based on aggregate inputs from the event store
func getEvents(ctx context.Context, eventStore core.EventStore, id, aggregateType string, fromVersion eventsourcing.Version) (*eventsourcing.Iterator, error) {
	// fetch events after the current version of the aggregate that could be fetched from the snapshot store
	eventIterator, err := eventStore.Get(ctx, id, aggregateType, core.Version(fromVersion))
	if err != nil {
		return nil, err
	}
	return &eventsourcing.Iterator{
		CoreIterator: eventIterator,
	}, nil
}
