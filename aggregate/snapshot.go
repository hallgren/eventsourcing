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

var encoderSnapshot eventsourcing.Encoder = eventsourcing.EncoderJSON{}

// SetEncoder sets the snapshot encoder
func SetEncoderSnapshot(e eventsourcing.Encoder) {
	encoderSnapshot = e
}

// LoadSnapshot build the aggregate based on its snapshot data not including its events.
// Beware that it could be more events that has happened after the snapshot was taken
func LoadSnapshot(ctx context.Context, ss core.SnapshotStore, id string, a aggregate) error {
	if reflect.ValueOf(a).Kind() != reflect.Ptr {
		return ErrAggregateNeedsToBeAPointer
	}
	err := getSnapshot(ctx, ss, id, a)
	if err != nil && errors.Is(err, core.ErrSnapshotNotFound) {
		return eventsourcing.ErrAggregateNotFound
	}
	return err
}

func getSnapshot(ctx context.Context, ss core.SnapshotStore, id string, a aggregate) error {
	s, err := ss.Get(ctx, id, aggregateType(a))
	if err != nil {
		return err
	}

	// Does the aggregate have specific snapshot handling
	sa, ok := a.(snapshot)
	if ok {
		err = sa.DeserializeSnapshot(encoderSnapshot.Deserialize, s.State)
		if err != nil {
			return err
		}
	} else {
		err = encoderSnapshot.Deserialize(s.State, a)
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
func SaveSnapshot(ss core.SnapshotStore, a aggregate) error {
	root := a.root()
	if len(root.Events()) > 0 {
		return ErrUnsavedEvents
	}

	state := []byte{}
	var err error
	// Does the aggregate have specific snapshot handling
	sa, ok := a.(snapshot)
	if ok {
		state, err = sa.SerializeSnapshot(encoderSnapshot.Serialize)
		if err != nil {
			return err
		}
	} else {
		state, err = encoderSnapshot.Serialize(a)
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

	return ss.Save(snapshot)
}
