package eventsourcing_test

import (
	"context"
	"testing"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	ss "github.com/hallgren/eventsourcing/snapshotstore/memory"
)

func TestSaveAndLoadAggregate(t *testing.T) {
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = eventsourcing.AggregateSave(es, person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	// make sure the global version is set to 1
	if person.GlobalVersion() != 1 {
		t.Fatalf("global version is: %d expected: 1", person.GlobalVersion())
	}

	twin := Person{}
	err = eventsourcing.AggregateLoad(context.Background(), es, person.ID(), &twin)
	if err != nil {
		t.Fatalf("could not get aggregate err: %v", err)
	}

	// Check internal aggregate version
	if person.Version() != twin.Version() {
		t.Fatalf("Wrong version org %q copy %q", person.Version(), twin.Version())
	}

	// Check person Name
	if person.Name != twin.Name {
		t.Fatalf("Wrong Name org %q copy %q", person.Name, twin.Name)
	}
}

func TestLoadAggregateFromSnapshot(t *testing.T) {
	es := memory.Create()
	ss := ss.Create()
	eventsourcing.AggregateRegister(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = eventsourcing.AggregateSave(es, person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	// store snapshot
	err = eventsourcing.SnapshotSave(ss, person)
	if err != nil {
		t.Fatal(err)
	}

	// add one more event to the person aggregate
	person.GrowOlder()
	err = eventsourcing.AggregateSave(es, person)

	// load person to person2 from snaphost and events
	person2 := &Person{}
	err = eventsourcing.AggregateLoadFromSnapshot(context.Background(), es, ss, person.ID(), person2)
	if err != nil {
		t.Fatal(err)
	}
	if person.Age != person2.Age {
		t.Fatalf("expected same age on person(%d) and person2(%d)", person.Age, person2.Age)
	}
}

func TestLoadNoneExistingAggregate(t *testing.T) {
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	p := Person{}
	err := eventsourcing.AggregateLoad(context.Background(), es, "none_existing", &p)
	if err != eventsourcing.ErrAggregateNotFound {
		t.Fatal("could not get aggregate")
	}
}
