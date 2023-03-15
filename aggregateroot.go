package eventsourcing

import (
	"errors"
	"reflect"
	"time"
)

// Version is the event version used in event.Version, event.GlobalVersion and aggregateRoot
type Version uint64

// AggregateRoot to be included into aggregates
type AggregateRoot[T any] struct {
	aggregateID            string
	aggregateVersion       Version
	aggregateGlobalVersion Version
	aggregateEvents        []Event[T]
}

const (
	emptyAggregateID = ""
)

// ErrAggregateAlreadyExists returned if the aggregateID is set more than one time
var ErrAggregateAlreadyExists = errors.New("its not possible to set ID on already existing aggregate")

// TrackChange is used internally by behaviour methods to apply a state change to
// the current instance and also track it in order that it can be persisted later.
func (ar *AggregateRoot[T]) TrackChange(a Aggregate[T], data T) {
	ar.TrackChangeWithMetadata(a, data, nil)
}

// TrackChangeWithMetadata is used internally by behaviour methods to apply a state change to
// the current instance and also track it in order that it can be persisted later.
// meta data is handled by this func to store none related application state
func (ar *AggregateRoot[T]) TrackChangeWithMetadata(a Aggregate[T], data T, metadata map[string]interface{}) {
	// This can be overwritten in the constructor of the aggregate
	if ar.aggregateID == emptyAggregateID {
		ar.aggregateID = idFunc()
	}

	name := reflect.TypeOf(a).Elem().Name()
	event := Event[T]{
		AggregateID:   ar.aggregateID,
		Version:       ar.nextVersion(),
		AggregateType: name,
		Timestamp:     time.Now().UTC(),
		Data:          data,
		Metadata:      metadata,
	}
	ar.aggregateEvents = append(ar.aggregateEvents, event)
	a.Transition(event)
}

// BuildFromHistory builds the aggregate state from events
func (ar *AggregateRoot[T]) BuildFromHistory(a Aggregate[T], events []Event[T]) {
	for _, event := range events {
		a.Transition(event)
		//Set the aggregate ID
		ar.aggregateID = event.AggregateID
		// Make sure the aggregate is in the correct version (the last event)
		ar.aggregateVersion = event.Version
		ar.aggregateGlobalVersion = event.GlobalVersion
	}
}

func (ar *AggregateRoot[T]) setInternals(id string, version, globalVersion Version) {
	ar.aggregateID = id
	ar.aggregateVersion = version
	ar.aggregateGlobalVersion = globalVersion
	ar.aggregateEvents = []Event[T]{}
}

func (ar *AggregateRoot[T]) nextVersion() Version {
	return ar.Version() + 1
}

// update sets the AggregateVersion and AggregateGlobalVersion to the values in the last event
// This function is called after the aggregate is saved in the repository
func (ar *AggregateRoot[T]) update() {
	if len(ar.aggregateEvents) > 0 {
		lastEvent := ar.aggregateEvents[len(ar.aggregateEvents)-1]
		ar.aggregateVersion = lastEvent.Version
		ar.aggregateGlobalVersion = lastEvent.GlobalVersion
		ar.aggregateEvents = []Event[T]{}
	}
}

// path return the full name of the aggregate making it unique to other aggregates with
// the same name but placed in other packages.
func (ar *AggregateRoot[T]) path() string {
	return reflect.TypeOf(ar).Elem().PkgPath()
}

// SetID opens up the possibility to set manual aggregate ID from the outside
func (ar *AggregateRoot[T]) SetID(id string) error {
	if ar.aggregateID != emptyAggregateID {
		return ErrAggregateAlreadyExists
	}
	ar.aggregateID = id
	return nil
}

// ID returns the aggregate ID as a string
func (ar *AggregateRoot[T]) ID() string {
	return ar.aggregateID
}

// Root returns the included Aggregate Root state, and is used from the interface Aggregate.
func (ar *AggregateRoot[T]) Root() *AggregateRoot[T] {
	return ar
}

// Version return the version based on events that are not stored
func (ar *AggregateRoot[T]) Version() Version {
	if len(ar.aggregateEvents) > 0 {
		return ar.aggregateEvents[len(ar.aggregateEvents)-1].Version
	}
	return ar.aggregateVersion
}

// GlobalVersion returns the global version based on the last stored event
func (ar *AggregateRoot[T]) GlobalVersion() Version {
	return ar.aggregateGlobalVersion
}

// Events return the aggregate events from the aggregate
// make a copy of the slice preventing outsiders modifying events.
func (ar *AggregateRoot[T]) Events() []Event[T] {
	e := make([]Event[T], len(ar.aggregateEvents))
	copy(e, ar.aggregateEvents)
	return e
}

// UnsavedEvents return true if there's unsaved events on the aggregate
func (ar *AggregateRoot[T]) UnsavedEvents() bool {
	return len(ar.aggregateEvents) > 0
}
