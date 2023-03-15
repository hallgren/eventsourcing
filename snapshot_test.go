package eventsourcing_test

import (
	"context"
	"encoding/xml"
	"testing"

	memory2 "github.com/hallgren/eventsourcing/eventstore/memory"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/snapshotstore/memory"
	memsnap "github.com/hallgren/eventsourcing/snapshotstore/memory"
)

type snapshot struct {
	eventsourcing.AggregateRoot[Events]
	unexported string
	Exported   string
}

type Events interface{ events() }
type Event struct{}

func (e *Event) events() {}

type Event2 struct{}

func (e *Event2) events() {}

func New() *snapshot {
	s := snapshot{}
	s.TrackChange(&s, &Event{})
	return &s
}

func (s *snapshot) Command() {
	s.TrackChange(s, &Event2{})
}

func (s *snapshot) Transition(e eventsourcing.Event[Events]) {
	switch e.Data.(type) {
	case *Event:
		s.unexported = "unexported"
		s.Exported = "Exported"
	case *Event2:
		s.unexported = "unexported2"
		s.Exported = "Exported2"
	}
}

type snapshotInternal struct {
	UnExported string
	Exported   string
}

func (s *snapshot) Marshal(m eventsourcing.MarshalSnapshotFunc) ([]byte, error) {
	snap := snapshotInternal{
		UnExported: s.unexported,
		Exported:   s.Exported,
	}
	return m(snap)
}

func (s *snapshot) Unmarshal(m eventsourcing.UnmarshalSnapshotFunc, b []byte) error {
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
	ser := eventsourcing.NewSerializer[Events](xml.Marshal, xml.Unmarshal)
	s := eventsourcing.SnapshotNew(memory.New(), *ser)

	// use repo to reset events on person to be able to save snapshot
	repo := eventsourcing.NewRepository[Events](memory2.Create[Events](), s)

	snap := New()
	repo.Save(snap)
	err := s.Save(snap)
	if err != nil {
		t.Fatal(err)
	}

	snap.Command()
	repo.Save(snap)

	snap2 := snapshot{}
	err = repo.Get(snap.ID(), &snap2)
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

func TestSnapshot(t *testing.T) {
	ser := eventsourcing.NewSerializer[PersonEvent](xml.Marshal, xml.Unmarshal)
	s := eventsourcing.SnapshotNew(memory.New(), *ser)

	// use repo to reset events on person to be able to save snapshot
	repo := eventsourcing.NewRepository[PersonEvent](memory2.Create[PersonEvent](), s)

	person, err := CreatePersonWithID("123", "kalle")
	if err != nil {
		t.Fatal(err)
	}
	repo.Save(person)

	err = s.Save(person)
	if err != nil {
		t.Fatal(err)
	}

	p := Person{}
	err = s.Get(context.Background(), "123", &p)
	if err != nil {
		t.Fatalf("could not get person from snapshot %v", err)
	}
	if person.ID() != p.ID() {
		t.Fatalf("wrong ID in snapshot %q expected: %q", person.ID(), p.ID())
	}
	if p.Age != person.Age {
		t.Fatalf("wrong Age in snapshot %d expected: %d", p.Age, person.Age)
	}
	if p.ID() != person.ID() {
		t.Fatalf("wrong id %s %s", p.ID(), person.ID())
	}
	if p.Version() != person.Version() {
		t.Fatalf("wrong version %d %d", p.Version(), person.Version())
	}
	if p.GlobalVersion() != person.GlobalVersion() {
		t.Fatalf("wrong global version %d %d", p.GlobalVersion(), person.GlobalVersion())
	}

	// store the snapshot once more
	person.Age = 99
	s.Save(person)

	err = s.Get(context.Background(), person.ID(), &p)
	if err != nil {
		t.Fatalf("could not get snapshot %v", err)
	}
	if p.Age != person.Age {
		t.Fatalf("wrong age %d %d", p.Age, person.Age)
	}
}
func TestGetNoneExistingSnapshot(t *testing.T) {
	ser := eventsourcing.NewSerializer[Events](xml.Marshal, xml.Unmarshal)
	s := eventsourcing.SnapshotNew(memsnap.New(), *ser)
	p := Person{}
	err := s.Get(context.Background(), "noneExistingID", &p)
	if err == nil {
		t.Fatalf("could get none existing snapshot %v", err)
	}
}

func TestSaveEmptySnapshotID(t *testing.T) {
	ser := eventsourcing.NewSerializer[Events](xml.Marshal, xml.Unmarshal)
	s := eventsourcing.SnapshotNew(memsnap.New(), *ser)
	p := Person{}
	err := s.Save(&p)
	if err == nil {
		t.Fatalf("could save blank snapshot id %v", err)
	}
}
