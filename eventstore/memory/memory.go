package memory

import (
	"context"
	"sync"

	"github.com/hallgren/eventsourcing/core"
)

// Memory is a handler for event streaming
type Memory struct {
	aggregateEvents map[string][]core.Event // The memory structure where we store aggregate events
	eventsInOrder   []core.Event            // The global event order
	lock            sync.Mutex
}

// Create in memory event store
func Create() *Memory {
	return &Memory{
		aggregateEvents: make(map[string][]core.Event),
		eventsInOrder:   make([]core.Event, 0),
	}
}

// Save an aggregate (its events)
func (e *Memory) Save(events []core.Event) error {
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
	currentVersion := core.Version(0)

	if len(evBucket) > 0 {
		// Last version in the list
		lastEvent := evBucket[len(evBucket)-1]
		currentVersion = lastEvent.Version
	}

	// Make sure no other has saved event to the same aggregate concurrently
	if core.Version(currentVersion)+1 != events[0].Version {
		return core.ErrConcurrency
	}

	for i, event := range events {
		// set the global version on the event +1 as if the event was already on the eventsInOrder slice
		event.GlobalVersion = core.Version(len(e.eventsInOrder) + 1)
		evBucket = append(evBucket, event)
		e.eventsInOrder = append(e.eventsInOrder, event)
		// override the event in the slice exposing the GlobalVersion to the caller
		events[i].GlobalVersion = event.GlobalVersion
	}

	e.aggregateEvents[bucketName] = evBucket
	return nil
}

// Get aggregate events
func (e *Memory) Get(ctx context.Context, id string, aggregateType string, afterVersion core.Version) (core.Iterator, error) {
	var events []core.Event
	// make sure its thread safe
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, e := range e.aggregateEvents[aggregateKey(aggregateType, id)] {
		if e.Version > afterVersion {
			events = append(events, e)
		}
	}
	return &iterator{events: events}, ctx.Err()
}

// Close does nothing
func (e *Memory) Close() {}

// aggregateKey generates a key to store events against from aggregateType and aggregateID
func aggregateKey(aggregateType, aggregateID string) string {
	return aggregateType + "_" + aggregateID
}

// globalEvents returns count events in order globally from the start position
func (e *Memory) globalEvents(start core.Version, count uint64) ([]core.Event, error) {
	events := make([]core.Event, 0, count)
	// make sure its thread safe
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, e := range e.eventsInOrder {
		// find start position and append until counter is 0
		if e.GlobalVersion >= start {
			events = append(events, e)
			count--
			if count == 0 {
				break
			}
		}
	}
	return events, nil
}

// All iterate over all events in GlobalEvents order
func (m *Memory) All(start core.Version, count uint64) func() (core.Iterator, error) {
	return func() (core.Iterator, error) {
		events, err := m.globalEvents(start, count)
		if err != nil {
			return nil, err
		}

		// no events to fetch
		if len(events) == 0 {
			return core.ZeroIterator{}, nil
		}

		// next time the function is called it will start from the last fetched event +1
		start = events[len(events)-1].GlobalVersion + 1
		return &iterator{events: events}, nil
	}
}
