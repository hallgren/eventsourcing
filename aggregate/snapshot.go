package aggregate

import (
	"context"
	"errors"
	"reflect"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
)

// ErrUnsavedEvents aggregate events must be saved before creating snapshot
var ErrUnsavedEvents = errors.New("aggregate holds unsaved events")

type SerializeFunc func(v interface{}) ([]byte, error)
type DeserializeFunc func(data []byte, v interface{}) error

// snapshot interface is used to serialize an aggregate that has properties that are not exported
type snapshot interface {
	SerializeSnapshot(SerializeFunc) ([]byte, error)
	DeserializeSnapshot(DeserializeFunc, []byte) error
}

var encoder eventsourcing.Encoder = eventsourcing.EncoderJSON{}

// SetEncoder sets the snapshot encoder
func SetEncoder(e eventsourcing.Encoder) {
	encoder = e
}

type SnapshotRepository struct {
	snapshotStore core.SnapshotStore
}

// NewSnapshotRepository factory function
func NewSnapshotRepository(snapshotStore core.SnapshotStore) *SnapshotRepository {
	return &SnapshotRepository{
		snapshotStore: snapshotStore,
	}
}

// GetSnapshot returns aggregate that is based on the snapshot data
// Beware that it could be more events that has happened after the snapshot was taken
func (sr *SnapshotRepository) Get(ctx context.Context, id string, a aggregate) error {
	if reflect.ValueOf(a).Kind() != reflect.Ptr {
		return ErrAggregateNeedsToBeAPointer
	}
	err := sr.getSnapshot(ctx, id, a)
	if err != nil && errors.Is(err, core.ErrSnapshotNotFound) {
		return eventsourcing.ErrAggregateNotFound
	}
	return err
}

func (sr *SnapshotRepository) getSnapshot(ctx context.Context, id string, a aggregate) error {
	s, err := sr.snapshotStore.Get(ctx, id, aggregateType(a))
	if err != nil {
		return err
	}

	// Does the aggregate have specific snapshot handling
	sa, ok := a.(snapshot)
	if ok {
		err = sa.DeserializeSnapshot(encoder.Deserialize, s.State)
		if err != nil {
			return err
		}
	} else {
		err = encoder.Deserialize(s.State, a)
		if err != nil {
			return err
		}
	}

	// set the internal aggregate properties
	root := a.root()
	root.aggregateGlobalVersion = eventsourcing.Version(s.GlobalVersion)
	root.aggregateVersion = eventsourcing.Version(s.Version)
	root.aggregateID = s.ID

	return nil
}

// SaveSnapshot will only store the snapshot and will return an error if there are events that are not stored
func (sr *SnapshotRepository) Save(a aggregate) error {
	root := a.root()
	if len(root.Events()) > 0 {
		return ErrUnsavedEvents
	}

	state := []byte{}
	var err error
	// Does the aggregate have specific snapshot handling
	sa, ok := a.(snapshot)
	if ok {
		state, err = sa.SerializeSnapshot(encoder.Serialize)
		if err != nil {
			return err
		}
	} else {
		state, err = encoder.Serialize(a)
		if err != nil {
			return err
		}
	}

	snapshot := core.Snapshot{
		ID:            root.ID(),
		Type:          aggregateType(a),
		Version:       core.Version(root.Version()),
		GlobalVersion: core.Version(root.GlobalVersion()),
		State:         state,
	}

	err = sr.snapshotStore.Save(snapshot)
	return err
}
