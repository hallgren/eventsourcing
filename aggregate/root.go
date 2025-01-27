package aggregate

import (
	"reflect"
	"time"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
)

// Root to be included into aggregates to give it the aggregate root behaviors
type Root struct {
	aggregateID            string
	aggregateVersion       eventsourcing.Version
	aggregateGlobalVersion eventsourcing.Version
	aggregateEvents        []eventsourcing.Event
}

const emptyAggregateID = ""

// TrackChange is used internally by behaviour methods to apply a state change to
// the current instance and also track it in order that it can be persisted later.
func TrackChange(a aggregate, data interface{}) {
	TrackChangeWithMetadata(a, data, nil)
}

// TrackChangeWithMetadata is used internally by behaviour methods to apply a state change to
// the current instance and also track it in order that it can be persisted later.
// meta data is handled by this func to store none related application state
func TrackChangeWithMetadata(a aggregate, data interface{}, metadata map[string]interface{}) {
	ar := a.root()
	// This can be overwritten in the constructor of the aggregate
	if ar.aggregateID == emptyAggregateID {
		ar.aggregateID = idFunc()
	}

	event := eventsourcing.NewEvent(
		core.Event{
			AggregateID:   ar.aggregateID,
			Version:       ar.nextVersion(),
			AggregateType: aggregateType(a),
			Timestamp:     time.Now().UTC(),
		},
		data,
		metadata,
	)
	ar.aggregateEvents = append(ar.aggregateEvents, event)
	a.Transition(event)
}

// buildFromHistory builds the aggregate state from events
func buildFromHistory(a aggregate, events []eventsourcing.Event) {
	root := a.root()
	for _, event := range events {
		a.Transition(event)
		//Set the aggregate ID
		root.aggregateID = event.AggregateID()
		// Make sure the aggregate is in the correct version (the last event)
		root.aggregateVersion = event.Version()
		root.aggregateGlobalVersion = event.GlobalVersion()
	}
}

func (ar *Root) nextVersion() core.Version {
	return core.Version(ar.Version()) + 1
}

// update sets the AggregateVersion and AggregateGlobalVersion to the values in the last event
// This function is called after the aggregate is saved in the repository
func (ar *Root) update() {
	if len(ar.aggregateEvents) > 0 {
		lastEvent := ar.aggregateEvents[len(ar.aggregateEvents)-1]
		ar.aggregateVersion = lastEvent.Version()
		ar.aggregateGlobalVersion = lastEvent.GlobalVersion()
		ar.aggregateEvents = []eventsourcing.Event{}
	}
}

// path return the full name of the aggregate making it unique to other aggregates with
// the same name but placed in other packages.
func (ar *Root) path() string {
	return reflect.TypeOf(ar).Elem().PkgPath()
}

// SetID opens up the possibility to set manual aggregate ID from the outside
func (ar *Root) SetID(id string) error {
	if ar.aggregateID != emptyAggregateID {
		return eventsourcing.ErrAggregateAlreadyExists
	}
	ar.aggregateID = id
	return nil
}

// ID returns the aggregate ID as a string
func (ar *Root) ID() string {
	return ar.aggregateID
}

// root returns the included Aggregate Root state, and is used from the interface Aggregate.
func (ar *Root) root() *Root {
	return ar
}

// Version return the version based on events that are not stored
func (ar *Root) Version() eventsourcing.Version {
	if len(ar.aggregateEvents) > 0 {
		return ar.aggregateEvents[len(ar.aggregateEvents)-1].Version()
	}
	return eventsourcing.Version(ar.aggregateVersion)
}

// GlobalVersion returns the global version based on the last stored event
func (ar *Root) GlobalVersion() eventsourcing.Version {
	return eventsourcing.Version(ar.aggregateGlobalVersion)
}

// Events return the aggregate events from the aggregate
// make a copy of the slice preventing outsiders modifying events.
func (ar *Root) Events() []eventsourcing.Event {
	e := make([]eventsourcing.Event, len(ar.aggregateEvents))
	// convert internal event to external event
	for i, event := range ar.aggregateEvents {
		e[i] = event
	}
	return e
}

// UnsavedEvents return true if there's unsaved events on the aggregate
func (ar *Root) UnsavedEvents() bool {
	return len(ar.aggregateEvents) > 0
}

func aggregateType(a interface{}) string {
	return reflect.TypeOf(a).Elem().Name()
}
