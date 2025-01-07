package eventsourcing

import (
	"context"
	"errors"
	"fmt"

	"github.com/hallgren/eventsourcing/core"
)

// Aggregate interface to use the aggregate root specific methods
type aggregate interface {
	root() *AggregateRoot
	Transition(event Event)
	Register(RegisterFunc)
}

type EventSubscribers interface {
	All(f func(e Event)) *subscription
	Event(f func(e Event), events ...interface{}) *subscription
	Name(f func(e Event), aggregate string, events ...string) *subscription
}

type encoder interface {
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
	eventStream *EventStream
	eventStore  core.EventStore
	// register that convert the Data []byte to correct type
	register *Register
	// encoder to serialize / deserialize events
	encoder     encoder
	Projections *ProjectionHandler
}

// NewRepository factory function
func NewEventRepository(eventStore core.EventStore) *EventRepository {
	register := NewRegister()
	encoder := EncoderJSON{}

	return &EventRepository{
		eventStore:  eventStore,
		eventStream: NewEventStream(),
		register:    register,
		encoder:     encoder, // Default to JSON encoder
		Projections: NewProjectionHandler(register, encoder),
	}
}

// Encoder change the default JSON encoder that serializer/deserializer events
func (er *EventRepository) Encoder(e encoder) {
	// set encoder on event repository
	er.encoder = e
	// set encoder in projection handler
	er.Projections.Encoder = e
}

func (er *EventRepository) Register(a aggregate) {
	er.register.Register(a)
}

// Subscribers returns an interface with all event subscribers
func (er *EventRepository) Subscribers() EventSubscribers {
	return er.eventStream
}

func (er *EventRepository) AggregateEvents(ctx context.Context, id, aggregateType string, fromVersion Version) ([]Event, error) {
	var events = make([]Event, 0)
	// fetch events after the current version of the aggregate that could be fetched from the snapshot store
	eventIterator, err := er.eventStore.Get(ctx, id, aggregateType, core.Version(fromVersion))
	if err != nil {
		return nil, err
	}
	defer eventIterator.Close()

	for eventIterator.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			event, err := eventIterator.Value()
			if err != nil {
				return nil, err
			}
			// apply the event to the aggregate
			f, found := er.register.EventRegistered(event)
			if !found {
				continue
			}
			data := f()
			err = er.encoder.Deserialize(event.Data, &data)
			if err != nil {
				return nil, err
			}
			metadata := make(map[string]interface{})
			err = er.encoder.Deserialize(event.Metadata, &metadata)
			if err != nil {
				return nil, err
			}

			events = append(events, NewEvent(event, data, metadata))
		}
	}

	if len(events) == 0 {
		return nil, ErrAggregateNotFound
	}
	return events, nil
}

func (er *EventRepository) Save(events []Event) (Version, error) {
	var esEvents = make([]core.Event, 0, len(events))

	for _, event := range events {
		data, err := er.encoder.Serialize(event.Data())
		if err != nil {
			return 0, err
		}
		metadata, err := er.encoder.Serialize(event.Metadata())
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
		_, ok := er.register.EventRegistered(esEvent)
		if !ok {
			return 0, ErrEventNotRegistered
		}
		esEvents = append(esEvents, esEvent)
	}

	err := er.eventStore.Save(esEvents)
	if err != nil {
		if errors.Is(err, core.ErrConcurrency) {
			return 0, ErrConcurrency
		}
		return 0, fmt.Errorf("error from event store: %w", err)
	}
	// publish the saved events to subscribers
	er.eventStream.Publish(events)

	return Version(esEvents[len(esEvents)-1].GlobalVersion), nil
}
