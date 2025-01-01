package eventsourcing

import (
	"errors"
	"reflect"
	"time"

	"github.com/hallgren/eventsourcing/core"
)

// ErrAggregateAlreadyExists returned if the aggregateID is set more than one time
var ErrAggregateAlreadyExists = errors.New("its not possible to set ID on already existing aggregate")

// ErrAggregateNeedsToBeAPointer return if aggregate is sent in as value object
var ErrAggregateNeedsToBeAPointer = errors.New("aggregate needs to be a pointer")

// Aggregate interface to use the aggregate root specific methods
type aggregate interface {
	root() *AggregateRoot
	Transition(event Event)
	Register(RegisterFunc)
}

// AggregateAddEvent is used internally by behaviour methods to apply a state change to
// the current instance and also track it in order that it can be persisted later.
func AggregateAddEvent(a aggregate, data interface{}) {
	AggregateAddEventWithMetadata(a, data, nil)
}

// AggregateAddEventWithMetadata is used internally by behaviour methods to apply a state change to
// the current instance and also track it in order that it can be persisted later.
// meta data is handled by this func to store none related application state
func AggregateAddEventWithMetadata(a aggregate, data interface{}, metadata map[string]interface{}) {
	ar := root(a)
	// This can be overwritten in the constructor of the aggregate
	if ar.aggregateID == emptyAggregateID {
		ar.aggregateID = idFunc()
	}

	event := Event{
		event: core.Event{
			AggregateID:   ar.aggregateID,
			Version:       ar.nextVersion(),
			AggregateType: aggregateType(a),
			Timestamp:     time.Now().UTC(),
		},
		data:     data,
		metadata: metadata,
	}
	ar.aggregateEvents = append(ar.aggregateEvents, event)
	a.Transition(event)
}

// buildFromHistory builds the aggregate state from events
func buildFromHistory(a aggregate, events []Event) {
	ar := root(a)
	for _, event := range events {
		a.Transition(event)
		//Set the aggregate ID
		ar.aggregateID = event.AggregateID()
		// Make sure the aggregate is in the correct version (the last event)
		ar.aggregateVersion = event.Version()
		ar.aggregateGlobalVersion = event.GlobalVersion()
	}
}

// AggregateSetID opens up the possibility to set manual aggregate ID from the outside
func AggregateSetID(a aggregate, id string) error {
	ar := root(a)
	if ar.aggregateID != emptyAggregateID {
		return ErrAggregateAlreadyExists
	}
	ar.aggregateID = id
	return nil
}

// AggregateID returns the identifier of the aggregate
func AggregateID(a aggregate) string {
	return root(a).aggregateID
}

// root gets the underlaying AggregateRoot
func root(a aggregate) *AggregateRoot {
	return a.root()
}

// AggregateVersion is the internal aggregate version
func AggregateVersion(a aggregate) Version {
	return root(a).version()
}

// AggregateGlobalVersion returns the global version based on the last stored event
func AggregateGlobalVersion(a aggregate) Version {
	ar := root(a)
	return Version(ar.aggregateGlobalVersion)
}

// AggregateEvents return the aggregate events from the aggregate
// make a copy of the slice preventing outsiders modifying events.
func AggregateEvents(a aggregate) []Event {
	ar := root(a)
	e := make([]Event, len(ar.aggregateEvents))
	// convert internal event to external event
	for i, event := range ar.aggregateEvents {
		e[i] = event
	}
	return e
}

// AggregateUnsavedEvents return true if there's unsaved events on the aggregate
func AggregateUnsavedEvents(a aggregate) bool {
	ar := root(a)
	return len(ar.aggregateEvents) > 0
}

func aggregateType(a aggregate) string {
	return reflect.TypeOf(a).Elem().Name()
}
