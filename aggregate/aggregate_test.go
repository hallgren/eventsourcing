package aggregate_test

import (
	"context"
	"testing"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	ss "github.com/hallgren/eventsourcing/snapshotstore/memory"
)

func TestSaveAndLoadAggregate(t *testing.T) {
	es := memory.Create()
	aggregate.Register(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = aggregate.Save(es, person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	// make sure the global version is set to 1
	if person.GlobalVersion() != 1 {
		t.Fatalf("global version is: %d expected: 1", person.GlobalVersion())
	}

	twin := Person{}
	err = aggregate.Load(context.Background(), es, person.ID(), &twin)
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
	aggregate.Register(&Person{})
	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = aggregate.Save(es, person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	// store snapshot
	err = aggregate.SaveSnapshot(ss, person)
	if err != nil {
		t.Fatal(err)
	}

	// add one more event to the person aggregate
	person.GrowOlder()
	err = aggregate.Save(es, person)

	// load person to person2 from snaphost and events
	person2 := &Person{}
	err = aggregate.LoadFromSnapshot(context.Background(), es, ss, person.ID(), person2)
	if err != nil {
		t.Fatal(err)
	}
	if person.Age != person2.Age {
		t.Fatalf("expected same age on person(%d) and person2(%d)", person.Age, person2.Age)
	}
}

func TestLoadNoneExistingAggregate(t *testing.T) {
	es := memory.Create()
	aggregate.Register(&Person{})

	p := Person{}
	err := aggregate.Load(context.Background(), es, "none_existing", &p)
	if err != eventsourcing.ErrAggregateNotFound {
		t.Fatal("could not get aggregate")
	}
}

func TestPostSaveTrigger(t *testing.T) {
	var trigger bool
	var event eventsourcing.Event
	es := memory.Create()
	aggregate.Register(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = aggregate.Save(es, person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}
	if trigger {
		t.Fatal("post trigger should not be activated")
	}

	// set the post save trigger function
	aggregate.SaveHook(func(events []eventsourcing.Event) {
		trigger = true
		event = events[0]
	}, &Person{})

	person.GrowOlder()
	err = aggregate.Save(es, person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}
	if !trigger {
		t.Fatal("post trigger should be activated")
	}
	if event.Reason() != "AgedOneYear" {
		t.Fatalf("expected AgedOneYear got %v", event.Reason())
	}
}
