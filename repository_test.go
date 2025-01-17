package eventsourcing_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/eventstore/memory"
)

// Person aggregate
type Person struct {
	aggregate.Root
	Name string
	Age  int
	Dead int
}

// Born event
type Born struct {
	Name string
}

// AgedOneYear event
type AgedOneYear struct {
}

// Register bind the events to the repository when the aggregate is registered.
func (person *Person) Register(f eventsourcing.RegisterFunc) {
	f(&Born{}, &AgedOneYear{})
}

// Transition the person state dependent on the events
func (person *Person) Transition(event eventsourcing.Event) {
	switch e := event.Data().(type) {
	case *Born:
		person.Age = 0
		person.Name = e.Name
	case *AgedOneYear:
		person.Age += 1
	}
}

// CreatePerson constructor for the Person
func CreatePerson(name string) (*Person, error) {
	if name == "" {
		return nil, errors.New("name can't be blank")
	}
	person := Person{}
	person.TrackChange(&person, &Born{Name: name})
	return &person, nil
}

// GrowOlder command
func (person *Person) GrowOlder() {
	metaData := make(map[string]interface{})
	metaData["foo"] = "bar"
	person.TrackChangeWithMetadata(person, &AgedOneYear{}, metaData)
}

func events() []eventsourcing.Event {
	events := make([]eventsourcing.Event, 0)
	events = append(events, eventsourcing.NewEvent(core.Event{
		AggregateID:   "123",
		Version:       1,
		GlobalVersion: 0,
		AggregateType: "Person",
		Timestamp:     time.Now().UTC(),
		Reason:        "Born",
		Data:          []byte{},
		Metadata:      []byte{},
	}, &Born{
		Name: "Kalle",
	}, nil))
	return events
}

func TestGetWithContext(t *testing.T) {
	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&Person{})
	e := events()
	_, err := repo.Save(events())
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	events, err := repo.AggregateEvents(context.Background(), "123", "Person", 0)
	if err != nil {
		t.Fatal("could not get aggregate")
	}

	reflect.DeepEqual(events, e)
}

func TestGetWithContextCancel(t *testing.T) {
	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.Save(person.Events())
	if err != nil {
		t.Fatal("could not save aggregate")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// cancel the context
	cancel()
	_, err = repo.AggregateEvents(ctx, person.ID(), "Person", 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error context.Canceled but was %v", err)
	}
}

func TestSubscriptionAllEvent(t *testing.T) {
	counter := 0
	f := func(e eventsourcing.Event) {
		counter++
	}
	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&Person{})

	es := eventsourcing.NewEventStream(repo)
	s := es.All(f)
	defer s.Close()

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	person.GrowOlder()
	person.GrowOlder()
	person.GrowOlder()

	_, err = repo.Save(person.Events())
	if err != nil {
		t.Fatal("could not save aggregate")
	}
	if counter != 4 {
		t.Errorf("No global events was received from the stream, got %q", counter)
	}
}

/*
func TestSubscriptionSpecificEvent(t *testing.T) {
	counter := 0
	f := func(e eventsourcing.Event) {
		counter++
	}
	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&Person{})

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
	f := func(e eventsourcing.Event) {
		counter++
	}
	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&Person{})

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
	f := func(e eventsourcing.Event) {
		counter++
	}
	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&Person{})

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
	var eventChanErr error

	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&Person{})

	// eventChan can hold 5 events before it get full and blocks.
	eventChan := make(chan eventsourcing.Event, 5)
	doneChan := make(chan struct{})
	f := func(e eventsourcing.Event) {
		eventChan <- e
	}

	// for every AgedOnYear create a new person and make it grow one year older
	go func() {
		defer close(doneChan)
		for e := range eventChan {
			switch e.Data().(type) {
			case *AgedOneYear:
				person, err := CreatePerson("kalle")
				if err != nil {
					eventChanErr = err
					return
				}
				person.GrowOlder()
				repo.Save(person)
			}
		}
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
	s2 := repo.Subscribers().All(func(e eventsourcing.Event) {
		switch e.Data().(type) {
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
	if eventChanErr != nil {
		t.Fatal(eventChanErr)
	}
	if ageCounter != 6 {
		t.Errorf("wrong number in ageCounter expected 6, got %v", ageCounter)
	}
}

func TestSaveWhenAggregateNotRegistered(t *testing.T) {
	repo := eventsourcing.NewEventRepository(memory.Create())

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if !errors.Is(err, eventsourcing.ErrAggregateNotRegistered) {
		t.Fatalf("could save aggregate that was not registered, err: %v", err)
	}
}

func TestSaveWhenEventNotRegistered(t *testing.T) {
	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&PersonNoRegisterEvents{})

	person, err := CreatePersonNoRegisteredEvents("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if !errors.Is(err, eventsourcing.ErrEventNotRegistered) {
		t.Fatalf("could save aggregate events that was not registered, err: %v", err)
	}
}

func TestMultipleSave(t *testing.T) {
	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	version := person.Version()

	err = repo.Save(person)
	if err != nil {
		t.Fatalf("save should be a nop, err: %v", err)
	}

	if version != person.Version() {
		t.Fatalf("the nop save should not change the aggregate version exp:%d, actual:%d", version, person.Version())
	}
}

// Person aggregate
type PersonNoRegisterEvents struct {
	eventsourcing.AggregateRoot
	Name string
	Age  int
	Dead int
}

// Born event
type BornNoRegisteredEvents struct {
	Name string
}

// CreatePersonNoRegisteredEvents constructor for the PersonNoRegisteredEvents
func CreatePersonNoRegisteredEvents(name string) (*PersonNoRegisterEvents, error) {
	if name == "" {
		return nil, errors.New("name can't be blank")
	}
	person := PersonNoRegisterEvents{}
	person.TrackChange(&person, &BornNoRegisteredEvents{Name: name})
	return &person, nil
}

// Transition the person state dependent on the events
func (person *PersonNoRegisterEvents) Transition(event eventsourcing.Event) {
	switch e := event.Data().(type) {
	case *BornNoRegisteredEvents:
		person.Age = 0
		person.Name = e.Name
	}
}

// Register bind the events to the repository when the aggregate is registered.
func (person *PersonNoRegisterEvents) Register(f eventsourcing.RegisterFunc) {
}

func TestProjectionFromRepo(t *testing.T) {
	es := memory.Create()
	repo := eventsourcing.NewEventRepository(es)
	repo.Register(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if err != nil {
		t.Fatal(err)
	}

	var projectedName string

	p := repo.Projections.Projection(es.All(0, 1), func(event eventsourcing.Event) error {
		switch e := event.Data().(type) {
		case *Born:
			projectedName = e.Name
		}
		return nil
	})
	p.RunOnce()

	if projectedName != "kalle" {
		t.Fatalf("expected projectedName to be kalle was %q", projectedName)
	}
}

func TestConcurrentRead(t *testing.T) {
	repo := eventsourcing.NewEventRepository(memory.Create())
	repo.Register(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}
	person2, err := CreatePerson("anka")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Save(person2)
	if err != nil {
		t.Fatal("could not save aggregate")
	}

	for i := 1; i <= 1000; i++ {
		p1 := Person{}
		p2 := Person{}
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			repo.Get(person.ID(), &p1)
			wg.Done()
		}()
		go func() {
			repo.Get(person2.ID(), &p2)
			wg.Done()
		}()
		wg.Wait()
		if p1.Name == p2.Name {
			t.Fatal("name should differ")
		}
	}
}
*/
