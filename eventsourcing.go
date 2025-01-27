package eventsourcing

import (
	"context"
	"errors"
	"fmt"

	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/internal"
)

var (
	// ErrAggregateNotFound returns if events not found for aggregate or aggregate was not based on snapshot from the outside
	ErrAggregateNotFound = errors.New("aggregate not found")

	// ErrAggregateNotRegistered when saving aggregate when it's not registered in the repository
	ErrAggregateNotRegistered = errors.New("aggregate not registered")

	// ErrEventNotRegistered when saving aggregate and one event is not registered in the repository
	ErrEventNotRegistered = errors.New("event not registered")

	// ErrConcurrency when the currently saved version of the aggregate differs from the new events
	ErrConcurrency = errors.New("concurrency error")

	// ErrAggregateAlreadyExists returned if the aggregateID is set more than one time
	ErrAggregateAlreadyExists = errors.New("its not possible to set ID on already existing aggregate")

	// ErrAggregateNeedsToBeAPointer return if aggregate is sent in as value object
	ErrAggregateNeedsToBeAPointer = errors.New("aggregate needs to be a pointer")

	// ErrUnsavedEvents aggregate events must be saved before creating snapshot
	ErrUnsavedEvents = errors.New("aggregate holds unsaved events")
)

type Encoder interface {
	Serialize(v interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}

// Encoder change the default JSON encoder that serializer/deserializer events
func SetEventEncoder(e Encoder) {
	internal.EventEncoder = e
}

// SetSnapshotEncoder sets the snapshot encoder
func SetSnapshotEncoder(e Encoder) {
	internal.SnapshotEncoder = e
}

var publisherFunc = func(events []Event) {}

// GetEvents return event iterator based on aggregate inputs from the event store
func GetEvents(ctx context.Context, eventStore core.EventStore, id, aggregateType string, fromVersion Version) (*Iterator, error) {
	// fetch events after the current version of the aggregate that could be fetched from the snapshot store
	eventIterator, err := eventStore.Get(ctx, id, aggregateType, core.Version(fromVersion))
	if err != nil {
		return nil, err
	}
	return &Iterator{
		iterator: eventIterator,
	}, nil
}

// Save events to the event store
func SaveEvents(eventStore core.EventStore, events []Event) (Version, error) {
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
			return 0, fmt.Errorf("%s %w", esEvent.Reason, ErrEventNotRegistered)
		}
		esEvents = append(esEvents, esEvent)
	}

	err := eventStore.Save(esEvents)
	if err != nil {
		if errors.Is(err, core.ErrConcurrency) {
			return 0, ErrConcurrency
		}
		return 0, fmt.Errorf("error from event store: %w", err)
	}
	// publish the saved events to subscribers
	publisherFunc(events)

	return Version(esEvents[len(esEvents)-1].GlobalVersion), nil
}
