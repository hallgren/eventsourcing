package eventsourcing

import (
	"reflect"

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

func (ar *AggregateRoot) nextVersion() core.Version {
	return core.Version(ar.version()) + 1
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

// ID returns the aggregate ID as a string
func (ar *AggregateRoot) id() string {
	return ar.aggregateID
}

// Version return the version based on events that are not stored
func (ar *AggregateRoot) version() Version {
	if len(ar.aggregateEvents) > 0 {
		return ar.aggregateEvents[len(ar.aggregateEvents)-1].Version()
	}
	return Version(ar.aggregateVersion)
}
