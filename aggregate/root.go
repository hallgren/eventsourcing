package aggregate

import (
	"reflect"
	"sync"
	"time"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
)

// Root to be included into aggregates to give it the aggregate root behaviors
type Root struct {
	id            string
	version       eventsourcing.Version
	globalVersion eventsourcing.Version
	events        []eventsourcing.Event
}

const emptyID = ""

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
	if ar.id == emptyID {
		ar.id = idFunc()
	}

	event := eventsourcing.NewEvent(
		core.Event{
			AggregateID:   ar.id,
			Version:       ar.nextVersion(),
			AggregateType: aggregateType(a),
			Timestamp:     time.Now().UTC(),
		},
		data,
		metadata,
	)
	ar.events = append(ar.events, event)
	a.Transition(event)
}

// buildFromHistory builds the aggregate state from a single historic event.
// It's called once per event while replaying an aggregate's history, so it
// intentionally avoids wrapping the event in a slice to prevent an allocation
// per replayed event.
func buildFromHistory(a aggregate, event eventsourcing.Event) {
	root := a.root()
	a.Transition(event)
	//Set the aggregate ID
	root.id = event.AggregateID()
	// Make sure the aggregate is in the correct version (the last event)
	root.version = event.Version()
	root.globalVersion = event.GlobalVersion()
}

func (ar *Root) nextVersion() core.Version {
	return core.Version(ar.Version()) + 1
}

// SetID opens up the possibility to set manual aggregate ID from the outside
func (ar *Root) SetID(id string) error {
	if ar.id != emptyID {
		return eventsourcing.ErrAggregateAlreadyExists
	}
	ar.id = id
	return nil
}

// ID returns the aggregate ID as a string
func (ar *Root) ID() string {
	return ar.id
}

// root returns the included Aggregate Root state, and is used from the interface Aggregate.
//
//nolint:unused
func (ar *Root) root() *Root {
	return ar
}

// Version return the version based on events that are not stored
func (ar *Root) Version() eventsourcing.Version {
	if len(ar.events) > 0 {
		return ar.events[len(ar.events)-1].Version()
	}
	return eventsourcing.Version(ar.version)
}

// GlobalVersion returns the global version based on the last stored event
func (ar *Root) GlobalVersion() eventsourcing.Version {
	return eventsourcing.Version(ar.globalVersion)
}

// Events return the aggregate events from the aggregate
// make a copy of the slice preventing outsiders modifying events.
func (ar *Root) Events() []eventsourcing.Event {
	e := make([]eventsourcing.Event, len(ar.events))
	copy(e, ar.events)
	return e
}

// UnsavedEvents return true if there's unsaved events on the aggregate
func (ar *Root) UnsavedEvents() bool {
	return len(ar.events) > 0
}

func aggregateType(a interface{}) string {
	t := reflect.TypeOf(a).Elem()
	if name, ok := aggregateTypeCache.Load(t); ok {
		return name.(string)
	}
	name := t.Name()
	aggregateTypeCache.Store(t, name)
	return name
}

// aggregateTypeCache caches the reflect-derived aggregate type name per
// concrete type, avoiding repeated reflection on every TrackChange/Load/Save call.
var aggregateTypeCache sync.Map
