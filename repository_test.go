package eventsourcing_test

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"testing"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	memsnap "github.com/hallgren/eventsourcing/snapshotstore/memory"
)

func TestSaveAndGetAggregate(t *testing.T) {
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}

	// make sure the global version is set to 1
	if person.GlobalVersion() != 1 {
		t.Fatalf("global version is: %d expected: 1", person.GlobalVersion())
	}

	twin := Person{}
	err = repo.Get(person.ID(), &twin)
	if err != nil {
		t.Fatal("could not get aggregate")
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

func TestGetWithContext(t *testing.T) {
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}

	twin := Person{}
	err = repo.GetWithContext(context.Background(), person.ID(), &twin)
	if err != nil {
		t.Fatal("could not get aggregate")
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

func TestGetWithContextCancel(t *testing.T) {
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}

	twin := Person{}
	ctx, cancel := context.WithCancel(context.Background())

	// cancel the context
	cancel()
	err = repo.GetWithContext(ctx, person.ID(), &twin)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error context.Canceled but was %v", err)
	}
}

func TestGetNoneExistingAggregate(t *testing.T) {
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)

	p := Person{}
	err := repo.Get("none_existing", &p)
	if err != eventsourcing.ErrAggregateNotFound {
		t.Fatal("could not get aggregate")
	}
}

func TestSaveAndGetAggregateSnapshotAndEvents(t *testing.T) {
	ser := eventsourcing.NewSerializer[PersonEvent](xml.Marshal, xml.Unmarshal)
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), eventsourcing.SnapshotNew[PersonEvent](memsnap.New(), *ser))

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}

	// save person to snapshot store
	err = repo.SaveSnapshot(person)
	if err != nil {
		t.Fatal(err)
	}
	person.GrowOlder()
	repo.Save(person)
	twin := Person{}
	err = repo.Get(person.ID(), &twin)
	if err != nil {
		t.Fatal("could not get aggregate")
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

func TestSaveAndGetAggregateSnapshotAndEventsWithContext(t *testing.T) {
	ser := eventsourcing.NewSerializer[PersonEvent](xml.Marshal, xml.Unmarshal)
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), eventsourcing.SnapshotNew(memsnap.New(), *ser))

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}

	// save person to snapshot store
	err = repo.SaveSnapshot(person)
	if err != nil {
		t.Fatal(err)
	}
	person.GrowOlder()
	repo.Save(person)
	twin := Person{}
	ctx, cancel := context.WithCancel(context.Background())
	err = repo.GetWithContext(ctx, person.ID(), &twin)
	if err != nil {
		t.Fatal("could not get aggregate")
	}

	// Check internal aggregate version
	if person.Version() != twin.Version() {
		t.Fatalf("Wrong version org %q copy %q", person.Version(), twin.Version())
	}

	// Check person Name
	if person.Name != twin.Name {
		t.Fatalf("Wrong Name org %q copy %q", person.Name, twin.Name)
	}

	// cancel the context
	cancel()
	err = repo.GetWithContext(ctx, person.ID(), &twin)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("should be error context canceled")
	}
}

func TestSaveSnapshotWithUnsavedEvents(t *testing.T) {
	ser := eventsourcing.NewSerializer[PersonEvent](json.Marshal, json.Unmarshal)
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), eventsourcing.SnapshotNew(memsnap.New(), *ser))

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	// save person to snapshot store
	err = repo.SaveSnapshot(person)
	if err == nil {
		t.Fatalf("should not save snapshot with unsaved events %v", err)
	}
}

func TestSaveSnapshotWithoutSnapshotStore(t *testing.T) {
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	// save person to snapshot store
	err = repo.SaveSnapshot(person)
	if err == nil {
		t.Fatal("should not save snapshot when there is no snapshot store")
	}
}

func TestSubscriptionAllEvent(t *testing.T) {
	counter := 0
	f := func(e eventsourcing.Event[PersonEvent]) {
		counter++
	}
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)
	s := repo.Subscribers().All(f)
	defer s.Close()

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	person.GrowOlder()
	person.GrowOlder()
	person.GrowOlder()

	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}
	if counter != 4 {
		t.Errorf("No global events was received from the stream, got %q", counter)
	}
}

func TestSubscriptionSpecificEvent(t *testing.T) {
	counter := 0
	f := func(e eventsourcing.Event[PersonEvent]) {
		counter++
	}
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)
	s := repo.Subscribers().Event(f, &Born{}, &AgedOneYear{})
	defer s.Close()

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	person.GrowOlder()
	person.GrowOlder()
	person.GrowOlder()

	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}
	if counter != 4 {
		t.Errorf("No global events was received from the stream, got %q", counter)
	}
}

func TestSubscriptionAggregate(t *testing.T) {
	counter := 0
	f := func(e eventsourcing.Event[PersonEvent]) {
		counter++
	}
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)
	s := repo.Subscribers().Aggregate(f, &Person{})
	defer s.Close()

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	person.GrowOlder()
	person.GrowOlder()
	person.GrowOlder()

	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}
	if counter != 4 {
		t.Errorf("No global events was received from the stream, got %q", counter)
	}
}

func TestSubscriptionSpecificAggregate(t *testing.T) {
	counter := 0
	f := func(e eventsourcing.Event[PersonEvent]) {
		counter++
	}
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	s := repo.Subscribers().AggregateID(f, person)
	defer s.Close()

	person.GrowOlder()
	person.GrowOlder()
	person.GrowOlder()

	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}
	if counter != 4 {
		t.Errorf("No global events was received from the stream, got %q", counter)
	}
}

func TestEventChainDoesNotHang(t *testing.T) {
	repo := eventsourcing.NewRepository[PersonEvent](memory.Create[PersonEvent](), nil)

	// eventChan can hold 5 events before it get full and blocks.
	eventChan := make(chan eventsourcing.Event[PersonEvent], 5)
	doneChan := make(chan struct{})
	f := func(e eventsourcing.Event[PersonEvent]) {
		eventChan <- e
	}

	// for every AgedOnYear create a new person and make it grow one year older
	go func() {
		for e := range eventChan {
			switch e.Data.(type) {
			case *AgedOneYear:
				person, err := CreatePerson("kalle")
				if err != nil {
					t.Fatal(err)
				}
				person.GrowOlder()
				repo.Save(person)
			}
		}
		close(doneChan)
	}()

	// create the initial person and setup event subscription on the specific person events
	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	s := repo.Subscribers().AggregateID(f, person)
	defer s.Close()

	// subscribe to all events and filter out AgedOneYear
	ageCounter := 0
	s2 := repo.Subscribers().All(func(e eventsourcing.Event[PersonEvent]) {
		switch e.Data.(type) {
		case *AgedOneYear:
			// will match three times on the initial person and one each on the resulting AgedOneYear event
			ageCounter++
		}
	})
	defer s2.Close()

	person.GrowOlder()
	person.GrowOlder()
	person.GrowOlder()

	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}
	close(eventChan)
	<-doneChan
	if ageCounter != 6 {
		t.Errorf("wrong number in ageCounter expected 6, got %v", ageCounter)
	}
}
