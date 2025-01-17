package eventsourcing

import (
	"context"
	"errors"
	"fmt"

	"github.com/hallgren/eventsourcing/core"
)

type EventSubscribers interface {
	All(f func(e Event)) *subscription
	Event(f func(e Event), events ...interface{}) *subscription
	Name(f func(e Event), aggregate string, events ...string) *subscription
}

type Encoder interface {
	Serialize(v interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}

var (
	// ErrAggregateNotFound returns if events not found for aggregate or aggregate was not based on snapshot from the outside
	ErrAggregateNotFound = errors.New("aggregate not found")

	// ErrAggregateNotRegistered when saving aggregate when it's not registered in the repository
	ErrAggregateNotRegistered = errors.New("aggregate not registered")

	// ErrEventNotRegistered when saving aggregate and one event is not registered in the repository
	ErrEventNotRegistered = errors.New("event not registered")

	// ErrConcurrency when the currently saved version of the aggregate differs from the new events
	ErrConcurrency = errors.New("concurrency error")
)

// EventRepository is the returned instance from the factory function
type EventRepository struct {
	eventStore core.EventStore
}

// global encoder used for events
var encoder Encoder = EncoderJSON{}

// Encoder change the default JSON encoder that serializer/deserializer events
func SetEncoder(e Encoder) {
	encoder = e
}

// global event register
var register = NewRegister()

// ResetRegsiter reset the event regsiter
func ResetRegsiter() {
	register = NewRegister()
}

var publisherFunc = func(events []Event) {}

// AggregateEvents return event iterator based on aggregate inputs from the event store
func AggregateEvents(ctx context.Context, eventStore core.EventStore, id, aggregateType string, fromVersion Version) (*Iterator, error) {
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
func Save(eventStore core.EventStore, events []Event) (Version, error) {
	var esEvents = make([]core.Event, 0, len(events))

	for _, event := range events {
		data, err := encoder.Serialize(event.Data())
		if err != nil {
			return 0, err
		}
		metadata, err := encoder.Serialize(event.Metadata())
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
		_, ok := register.EventRegistered(esEvent)
		if !ok {
			return 0, ErrEventNotRegistered
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

// Regsiter registers the aggregate in the register
func Register(a agg) {
	register.Register(a)
}
