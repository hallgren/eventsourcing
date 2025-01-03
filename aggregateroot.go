package eventsourcing

import (
	"errors"
	"reflect"
	"time"

	"github.com/hallgren/eventsourcing/core"
)

// AggregateRoot to be included into aggregates
type AggregateRoot struct {
	aggregateID            string
	aggregateVersion       Version
	aggregateGlobalVersion Version
	aggregateEvents        []Event
}

const (
	emptyAggregateID = ""
)

// ErrAggregateAlreadyExists returned if the aggregateID is set more than one time
var ErrAggregateAlreadyExists = errors.New("its not possible to set ID on already existing aggregate")

// ErrAggregateNeedsToBeAPointer return if aggregate is sent in as value object
var ErrAggregateNeedsToBeAPointer = errors.New("aggregate needs to be a pointer")

// TrackChange is used internally by behaviour methods to apply a state change to
// the current instance and also track it in order that it can be persisted later.
func TrackChange(a aggregate, data interface{}) {
	TrackChangeWithMetadata(a, data, nil)
}

// TrackChangeWithMetadata is used internally by behaviour methods to apply a state change to
// the current instance and also track it in order that it can be persisted later.
// meta data is handled by this func to store none related application state
func TrackChangeWithMetadata(a aggregate, data interface{}, metadata map[string]interface{}) {
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

func (ar *AggregateRoot) nextVersion() core.Version {
	return core.Version(ar.Version()) + 1
}

// update sets the AggregateVersion and AggregateGlobalVersion to the values in the last event
// This function is called after the aggregate is saved in the repository
func (ar *AggregateRoot) update() {
	if len(ar.aggregateEvents) > 0 {
		lastEvent := ar.aggregateEvents[len(ar.aggregateEvents)-1]
		ar.aggregateVersion = lastEvent.Version()
		ar.aggregateGlobalVersion = lastEvent.GlobalVersion()
		ar.aggregateEvents = []Event{}
	}
}

// path return the full name of the aggregate making it unique to other aggregates with
// the same name but placed in other packages.
func (ar *AggregateRoot) path() string {
	return reflect.TypeOf(ar).Elem().PkgPath()
}

// SetID opens up the possibility to set manual aggregate ID from the outside
func SetID(a aggregate, id string) error {
	ar := root(a)
	if ar.aggregateID != emptyAggregateID {
		return ErrAggregateAlreadyExists
	}
	ar.aggregateID = id
	return nil
}

func ID(a aggregate) string {
	return root(a).aggregateID
}

// ID returns the aggregate ID as a string
func (ar *AggregateRoot) id() string {
	return ar.aggregateID
}

// root returns the included Aggregate Root state, and is used from the interface Aggregate.
func root(a aggregate) *AggregateRoot {
	return a.root()
}

// Version return the version based on events that are not stored
func (ar *AggregateRoot) Version() Version {
	if len(ar.aggregateEvents) > 0 {
		return ar.aggregateEvents[len(ar.aggregateEvents)-1].Version()
	}
	return Version(ar.aggregateVersion)
}

func LocalVersion(a aggregate) Version {
	ar := root(a)
	return ar.Version()
}

// GlobalVersion returns the global version based on the last stored event
func GlobalVersion(a aggregate) Version {
	ar := root(a)
	return Version(ar.aggregateGlobalVersion)
}

// Events return the aggregate events from the aggregate
// make a copy of the slice preventing outsiders modifying events.
func Events(a aggregate) []Event {
	ar := root(a)
	e := make([]Event, len(ar.aggregateEvents))
	// convert internal event to external event
	for i, event := range ar.aggregateEvents {
		e[i] = event
	}
	return e
}

// UnsavedEvents return true if there's unsaved events on the aggregate
func UnsavedEvents(a aggregate) bool {
	ar := root(a)
	return len(ar.aggregateEvents) > 0
}

func aggregateType(a aggregate) string {
	return reflect.TypeOf(a).Elem().Name()
}
