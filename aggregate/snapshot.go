package aggregate

import (
	"context"
	"errors"
	"reflect"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/internal"
)

type SnapshotMarshal func(v interface{}) ([]byte, error)
type SnapshotUnmarshal func(data []byte, v interface{}) error

// snapshot interface is used to serialize an aggregate that has properties that are not exported
type snapshot interface {
	root() *Root
	SerializeSnapshot(f SnapshotMarshal) ([]byte, error)
	DeserializeSnapshot(f SnapshotUnmarshal, d []byte) error
}

type aggregateSnapshot interface {
	aggregate
	snapshot
}

// LoadSnapshot build the aggregate based on its snapshot data not including its events.
// Beware that it could be more events that has happened after the snapshot was taken
func LoadSnapshot(ctx context.Context, ss core.SnapshotStore, id string, s snapshot) error {
	if reflect.ValueOf(s).Kind() != reflect.Ptr {
		return eventsourcing.ErrAggregateNeedsToBeAPointer
	}
	err := getSnapshot(ctx, ss, id, s)
	if err != nil && errors.Is(err, core.ErrSnapshotNotFound) {
		return eventsourcing.ErrAggregateNotFound
	}
	return err
}

func getSnapshot(ctx context.Context, ss core.SnapshotStore, id string, s snapshot) error {
	snap, err := ss.Get(ctx, id, aggregateType(s))
	if err != nil {
		return err
	}

	err = s.DeserializeSnapshot(internal.SnapshotEncoder.Deserialize, snap.State)
	if err != nil {
		return err
	}

	// set the internal aggregate properties
	root := s.root()
	root.globalVersion = eventsourcing.Version(snap.GlobalVersion)
	root.version = eventsourcing.Version(snap.Version)
	root.id = snap.ID

	return nil
}

// SaveSnapshot will only store the snapshot and will return an error if there are events that are not stored
func SaveSnapshot(ss core.SnapshotStore, s snapshot) error {
	root := s.root()
	if len(root.Events()) > 0 {
		return eventsourcing.ErrUnsavedEvents
	}

	state := []byte{}
	var err error
	state, err = s.SerializeSnapshot(internal.SnapshotEncoder.Serialize)
	if err != nil {
		return err
	}

	snapshot := core.Snapshot{
		ID:            root.ID(),
		Type:          aggregateType(s),
		Version:       core.Version(root.Version()),
		GlobalVersion: core.Version(root.GlobalVersion()),
		State:         state,
	}

	return ss.Save(snapshot)
}
