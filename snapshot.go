package eventsourcing

import (
	"context"
	"errors"
	"reflect"
)

// ErrEmptyID indicates that the aggregate ID was empty
var ErrEmptyID = errors.New("aggregate id is empty")

// ErrUnsavedEvents aggregate events must be saved before creating snapshot
var ErrUnsavedEvents = errors.New("aggregate holds unsaved events")

// Snapshot holds current state of an aggregate
type Snapshot struct {
	ID            string
	Type          string
	State         []byte
	Version       Version
	GlobalVersion Version
}

// SnapshotAggregate is an Aggregate plus extra methods to help serialize into a snapshot
type SnapshotAggregate[T any] interface {
	Aggregate[T]
	Marshal(m MarshalSnapshotFunc) ([]byte, error)
	Unmarshal(m UnmarshalSnapshotFunc, b []byte) error
}

// SnapshotHandler gets and saves snapshots
type SnapshotHandler[T any] struct {
	snapshotStore SnapshotStore
	serializer    Serializer[T]
}

// SnapshotNew constructs a SnapshotHandler
func SnapshotNew[T any](ss SnapshotStore, ser Serializer[T]) *SnapshotHandler[T] {
	return &SnapshotHandler[T]{
		snapshotStore: ss,
		serializer:    ser,
	}
}

// Save transform an aggregate to a snapshot
func (s *SnapshotHandler[T]) Save(i interface{}) error {
	sa, ok := i.(SnapshotAggregate[T])
	if ok {
		return s.saveSnapshotAggregate(sa)
	}
	a, ok := i.(Aggregate[T])
	if ok {
		return s.saveAggregate(a)
	}
	return errors.New("not an aggregate")
}

func (s *SnapshotHandler[T]) saveSnapshotAggregate(sa SnapshotAggregate[T]) error {
	root := sa.Root()
	err := validate(*root)
	if err != nil {
		return err
	}
	typ := reflect.TypeOf(sa).Elem().Name()
	b, err := sa.Marshal(s.serializer.Marshal)
	if err != nil {
		return err
	}
	snap := Snapshot{
		ID:            root.ID(),
		Type:          typ,
		Version:       root.Version(),
		GlobalVersion: root.GlobalVersion(),
		State:         b,
	}
	return s.snapshotStore.Save(snap)
}

func (s *SnapshotHandler[T]) saveAggregate(sa Aggregate[T]) error {
	root := sa.Root()
	err := validate(*root)
	if err != nil {
		return err
	}
	typ := reflect.TypeOf(sa).Elem().Name()
	b, err := s.serializer.Marshal(sa)
	if err != nil {
		return err
	}
	snap := Snapshot{
		ID:            root.ID(),
		Type:          typ,
		Version:       root.Version(),
		GlobalVersion: root.GlobalVersion(),
		State:         b,
	}
	return s.snapshotStore.Save(snap)
}

// Get fetch a snapshot and reconstruct an aggregate
func (s *SnapshotHandler[T]) Get(ctx context.Context, id string, i interface{}) error {
	typ := reflect.TypeOf(i).Elem().Name()
	snap, err := s.snapshotStore.Get(ctx, id, typ)
	if err != nil {
		return err
	}
	switch a := i.(type) {
	case SnapshotAggregate[T]:
		err := a.Unmarshal(s.serializer.Unmarshal, snap.State)
		if err != nil {
			return err
		}
		root := a.Root()
		root.setInternals(snap.ID, snap.Version, snap.GlobalVersion)
	case Aggregate[T]:
		err = s.serializer.Unmarshal(snap.State, a)
		if err != nil {
			return err
		}
		root := a.Root()
		root.setInternals(snap.ID, snap.Version, snap.GlobalVersion)
	default:
		return errors.New("not an aggregate")
	}
	return nil
}

// validate make sure the aggregate is valid to be saved
func validate[T any](root AggregateRoot[T]) error {
	if root.ID() == "" {
		return ErrEmptyID
	}
	if root.UnsavedEvents() {
		return ErrUnsavedEvents
	}
	return nil
}
