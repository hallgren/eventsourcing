package memory

import (
	"context"
	"sync"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/eventstore"
)

// Memory is a handler for event streaming
type Memory[T any] struct {
	aggregateEvents map[string][]eventsourcing.Event[T] // The memory structure where we store aggregate events
	eventsInOrder   []eventsourcing.Event[T]            // The global event order
	lock            sync.Mutex
}

type iterator[T any] struct {
	events   []eventsourcing.Event[T]
	position int
}

func (i *iterator[T]) Next() (eventsourcing.Event[T], error) {
	if len(i.events) <= i.position {
		return eventsourcing.Event[T]{}, eventsourcing.ErrNoMoreEvents
	}
	event := i.events[i.position]
	i.position++
	return event, nil
}

func (i *iterator[T]) Close() {
	i.events = nil
	i.position = 0
}

// Create in memory event store
func Create[T any]() *Memory[T] {
	return &Memory[T]{
		aggregateEvents: make(map[string][]eventsourcing.Event[T]),
		eventsInOrder:   make([]eventsourcing.Event[T], 0),
	}
}

// Save an aggregate (its events)
func (e *Memory[T]) Save(events []eventsourcing.Event[T]) error {
	// Return if there is no events to save
	if len(events) == 0 {
		return nil
	}

	// make sure its thread safe
	e.lock.Lock()
	defer e.lock.Unlock()

	// get bucket name from first event
	aggregateType := events[0].AggregateType
	aggregateID := events[0].AggregateID
	bucketName := aggregateKey(aggregateType, aggregateID)

	evBucket := e.aggregateEvents[bucketName]
	currentVersion := eventsourcing.Version(0)

	if len(evBucket) > 0 {
		// Last version in the list
		lastEvent := evBucket[len(evBucket)-1]
		currentVersion = lastEvent.Version
	}

	//Validate events
	err := eventstore.ValidateEvents(aggregateID, currentVersion, events)
	if err != nil {
		return err
	}

	for i, event := range events {
		// set the global version on the event +1 as if the event was already on the eventsInOrder slice
		event.GlobalVersion = eventsourcing.Version(len(e.eventsInOrder) + 1)
		evBucket = append(evBucket, event)
		e.eventsInOrder = append(e.eventsInOrder, event)
		// override the event in the slice exposing the GlobalVersion to the caller
		events[i].GlobalVersion = event.GlobalVersion
	}

	e.aggregateEvents[bucketName] = evBucket
	return nil
}

// Get aggregate events
func (e *Memory[T]) Get(ctx context.Context, id string, aggregateType string, afterVersion eventsourcing.Version) (eventsourcing.EventIterator[T], error) {
	var events []eventsourcing.Event[T]
	// make sure its thread safe
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, e := range e.aggregateEvents[aggregateKey(aggregateType, id)] {
		if e.Version > afterVersion {
			events = append(events, e)
		}
	}
	if len(events) == 0 {
		return nil, eventsourcing.ErrNoEvents
	}
	return &iterator[T]{events: events}, nil
}

// GlobalEvents will return count events in order globally from the start posistion
func (e *Memory[T]) GlobalEvents(start, count uint64) ([]eventsourcing.Event[T], error) {
	var events []eventsourcing.Event[T]
	// make sure its thread safe
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, e := range e.eventsInOrder {
		// find start position and append until counter is 0
		if uint64(e.GlobalVersion) >= start {
			events = append(events, e)
			count--
			if count == 0 {
				break
			}
		}
	}
	return events, nil
}

// Close does nothing
func (e *Memory[T]) Close() {}

// aggregateKey generate a aggregate key to store events against from aggregateType and aggregateID
func aggregateKey(aggregateType, aggregateID string) string {
	return aggregateType + "_" + aggregateID
}
