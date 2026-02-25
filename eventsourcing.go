package eventsourcing

import (
	"errors"

	"github.com/hallgren/eventsourcing/internal"
)

var (
	// ErrAggregateNotFound returns if events not found for aggregate or aggregate was not based on snapshot from the outside
	ErrAggregateNotFound = errors.New("aggregate not found")

	// ErrAggregateNotRegistered when aggregate is not registered via the aggregate.Register method and can be returned by aggregate.Save or aggregate.SetSaveHook
	ErrAggregateNotRegistered = errors.New("aggregate not registered")

	// ErrEventNotRegistered when saving aggregate and one event is not registered in the repository
	ErrEventNotRegistered = errors.New("event not registered")

	// ErrConcurrency when the currently saved version of the aggregate differs from the new events
	ErrConcurrency = errors.New("concurrency error")

	// ErrAggregateAlreadyExists returned if the aggregateID is set more than one time
	ErrAggregateAlreadyExists = errors.New("its not possible to set ID on already existing aggregate")

	// ErrAggregateNeedsToBeAPointer return if aggregate is sent in as value object
	ErrAggregateNeedsToBeAPointer = errors.New("aggregate needs to be a pointer")

	// ErrUnsavedEvents aggregate events must be saved before creating snapshot
	ErrUnsavedEvents = errors.New("aggregate holds unsaved events")
)

// Encoder is the interface used to Serialize/Deserialize events and snapshots
type Encoder interface {
	Serialize(v interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}

// SetEventEncoder change the default JSON encoder that serialize/deserialize events
func SetEventEncoder(e Encoder) {
	internal.EventEncoder = e
}

// SetSnapshotEncoder change the default JSON encoder that seialize/deserialize snapshots
func SetSnapshotEncoder(e Encoder) {
	internal.SnapshotEncoder = e
}
