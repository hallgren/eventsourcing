package eventsourcing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	snap "github.com/hallgren/eventsourcing/snapshotstore/memory"
)

func createPerson() *Person {
	es := memory.Create()
	eventsourcing.Register(&Person{})
	person, err := CreatePerson("kalle")
	if err != nil {
		panic(err)
	}
	eventsourcing.Save(es, person)

	return person
}

func TestSaveAndGetSnapshot(t *testing.T) {
	snapshotStore := snap.Create()
	person := createPerson()
	err := eventsourcing.SaveSnapshot(snapshotStore, person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	twin := Person{}
	err = eventsourcing.GetSnapshot(context.Background(), snapshotStore, person.ID(), &twin)
	if err != nil {
		t.Fatalf("could not get aggregate, err: %v", err)
	}

	// Check internal aggregate version
	if person.Version() != twin.Version() {
		t.Fatalf("Wrong version org %q copy %q", person.Version(), twin.Version())
	}

	if person.ID() != twin.ID() {
		t.Fatalf("Wrong id org %q copy %q", person.ID(), twin.ID())
	}

	if person.Name != twin.Name {
		t.Fatalf("Wrong name org: %q copy %q", person.Name, twin.Name)
	}
}

func TestGetNoneExistingSnapshotOrEvents(t *testing.T) {
	snapshotStore := snap.Create()
	person := Person{}

	err := eventsourcing.GetSnapshot(context.Background(), snapshotStore, "none_existing_id", &person)
	if !errors.Is(err, eventsourcing.ErrAggregateNotFound) {
		t.Fatal("should get error when no snapshot or event stored for aggregate")
	}
}

func TestGetNoneExistingSnapshot(t *testing.T) {
	snapshotStore := snap.Create()

	person := Person{}
	err := eventsourcing.GetSnapshot(context.Background(), snapshotStore, "none_existing_id", &person)
	if !errors.Is(err, eventsourcing.ErrAggregateNotFound) {
		t.Fatal("should get error when no snapshot stored for aggregate")
	}
}

func TestSaveSnapshotWithUnsavedEvents(t *testing.T) {
	snapshotStore := snap.Create()

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = eventsourcing.SaveSnapshot(snapshotStore, person)
	if err == nil {
		t.Fatalf("should not be able to save snapshot with unsaved events")
	}
}

// test custom snapshot struct to handle non-exported properties on aggregate
type snapshot struct {
	eventsourcing.Root
	unexported string
	Exported   string
	// to be able to save the snapshot after events are added to it.
	repo core.EventStore
}

type Event struct{}
type Event2 struct{}

func New() *snapshot {
	es := memory.Create()
	eventsourcing.Register(&snapshot{})
	s := snapshot{}
	eventsourcing.TrackChange(&s, &Event{})
	s.repo = es
	eventsourcing.Save(es, &s)
	return &s
}

func (s *snapshot) Command() {
	eventsourcing.TrackChange(s, &Event2{})
	eventsourcing.Save(s.repo, s)
}

func (s *snapshot) Transition(e eventsourcing.Event) {
	switch e.Data().(type) {
	case *Event:
		s.unexported = "unexported"
		s.Exported = "Exported"
	case *Event2:
		s.unexported = "unexported2"
		s.Exported = "Exported2"
	}
}

// Register bind the events to the repository when the aggregate is registered.
func (s *snapshot) Register(f eventsourcing.RegisterFunc) {
	f(&Event{}, &Event2{})
}

type snapshotInternal struct {
	UnExported string
	Exported   string
}

func (s *snapshot) SerializeSnapshot(m eventsourcing.SerializeFunc) ([]byte, error) {
	snap := snapshotInternal{
		UnExported: s.unexported,
		Exported:   s.Exported,
	}
	return m(snap)
}

func (s *snapshot) DeserializeSnapshot(m eventsourcing.DeserializeFunc, b []byte) error {
	snap := snapshotInternal{}
	err := m(b, &snap)
	if err != nil {
		return err
	}
	s.unexported = snap.UnExported
	s.Exported = snap.Exported
	return nil
}

func TestSnapshotNoneExported(t *testing.T) {
	snapshotStore := snap.Create()

	snap := New()
	err := eventsourcing.SaveSnapshot(snapshotStore, snap)
	if err != nil {
		t.Fatal(err)
	}

	snap.Command()
	err = eventsourcing.SaveSnapshot(snapshotStore, snap)
	if err != nil {
		t.Fatal(err)
	}

	snap2 := snapshot{}
	err = eventsourcing.GetSnapshot(context.Background(), snapshotStore, snap.ID(), &snap2)
	if err != nil {
		t.Fatal(err)
	}

	if snap.unexported != snap2.unexported {
		t.Fatalf("none exported value differed %s %s", snap.unexported, snap2.unexported)
	}

	if snap.Exported != snap2.Exported {
		t.Fatalf("exported value differed %s %s", snap.Exported, snap2.Exported)
	}
}
