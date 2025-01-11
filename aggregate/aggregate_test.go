package aggregate_test

import (
	"context"
	"testing"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/eventstore/memory"
)

func TestSaveAndGet(t *testing.T) {
	repo := eventsourcing.NewEventRepository(memory.Create())
	aggrepo := aggregate.NewAggregateRepository(repo, nil)
	aggrepo.Register(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = aggrepo.Save(person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	// make sure the global version is set to 1
	if person.GlobalVersion() != 1 {
		t.Fatalf("global version is: %d expected: 1", person.GlobalVersion())
	}

	twin := Person{}
	err = aggrepo.Get(context.Background(), person.ID(), &twin)
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

func TestGetNoneExistingAggregate(t *testing.T) {
	repo := eventsourcing.NewEventRepository(memory.Create())
	aggrepo := aggregate.NewAggregateRepository(repo, nil)
	aggrepo.Register(&Person{})

	p := Person{}
	err := aggrepo.Get(context.Background(), "none_existing", &p)
	if err != eventsourcing.ErrAggregateNotFound {
		t.Fatal("could not get aggregate")
	}
}
