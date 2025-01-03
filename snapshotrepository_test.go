package eventsourcing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	snap "github.com/hallgren/eventsourcing/snapshotstore/memory"
)

func setupSnapshotRepository() *eventsourcing.SnapshotRepository {
	eventrepo := eventsourcing.NewEventRepository(memory.Create())
	eventrepo.Register(&Person{})

	return eventsourcing.NewSnapshotRepository(snap.Create(), eventrepo)
}

func TestSaveAndGetSnapshot(t *testing.T) {
	snapshotrepo := setupSnapshotRepository()

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = snapshotrepo.Save(person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	twin := Person{}
	err = snapshotrepo.GetWithContext(context.Background(), eventsourcing.Aggregate.ID(person), &twin)
	if err != nil {
		t.Fatal("could not get aggregate")
	}

	// Check internal aggregate version
	if eventsourcing.Aggregate.Version(person) != eventsourcing.Aggregate.Version(&twin) {
		t.Fatalf("Wrong version org %q copy %q", eventsourcing.Aggregate.Version(person), eventsourcing.Aggregate.Version(&twin))
	}

	if eventsourcing.Aggregate.ID(person) != eventsourcing.Aggregate.ID(&twin) {
		t.Fatalf("Wrong id org %q copy %q", eventsourcing.Aggregate.ID(person), eventsourcing.Aggregate.ID(&twin))
	}

	if person.Name != twin.Name {
		t.Fatalf("Wrong name org: %q copy %q", person.Name, twin.Name)
	}
}

func TestGetNoneExistingSnapshotOrEvents(t *testing.T) {
	snapshotrepo := setupSnapshotRepository()

	person := Person{}
	err := snapshotrepo.GetWithContext(context.Background(), "none_existing_id", &person)
	if !errors.Is(err, eventsourcing.ErrAggregateNotFound) {
		t.Fatal("should get error when no snapshot or event stored for aggregate")
	}
}

func TestGetNoneExistingSnapshot(t *testing.T) {
	snapshotrepo := setupSnapshotRepository()

	person := Person{}
	err := snapshotrepo.GetSnapshot(context.Background(), "none_existing_id", &person)
	if !errors.Is(err, eventsourcing.ErrAggregateNotFound) {
		t.Fatal("should get error when no snapshot stored for aggregate")
	}
}

func TestSaveSnapshotWithUnsavedEvents(t *testing.T) {
	snapshotrepo := setupSnapshotRepository()

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = snapshotrepo.SaveSnapshot(person)
	if err == nil {
		t.Fatalf("should not be able to save snapshot with unsaved events")
	}
}

// test custom snapshot struct to handle non-exported properties on aggregate

type snapshot struct {
	eventsourcing.AggregateRoot
	unexported string
	Exported   string
}

type Event struct{}
type Event2 struct{}

func New() *snapshot {
	s := snapshot{}
	eventsourcing.Aggregate.Add(&s, &Event{})
	return &s
}

func (s *snapshot) Command() {
	eventsourcing.Aggregate.Add(s, &Event2{})
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
	snapshotrepo := setupSnapshotRepository()
	snapshotrepo.Register(&snapshot{})

	snap := New()
	err := snapshotrepo.Save(snap)
	if err != nil {
		t.Fatal(err)
	}

	snap.Command()
	snapshotrepo.Save(snap)

	snap2 := snapshot{}
	err = snapshotrepo.GetWithContext(context.Background(), eventsourcing.Aggregate.ID(snap), &snap2)
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
