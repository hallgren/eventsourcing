package aggregate_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
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

// CreatePerson constructor for the Person
func CreatePerson(name string) (*Person, error) {
	if name == "" {
		return nil, errors.New("name can't be blank")
	}
	person := Person{}
	aggregate.TrackChange(&person, &Born{Name: name})
	return &person, nil
}

// CreatePersonWithID constructor for the Person that sets the aggregate ID from the outside
func CreatePersonWithID(id, name string) (*Person, error) {
	if name == "" {
		return nil, errors.New("name can't be blank")
	}

	person := Person{}
	err := person.SetID(id)
	if err == eventsourcing.ErrAggregateAlreadyExists {
		return nil, err
	} else if err != nil {
		return nil, err
	}
	aggregate.TrackChange(&person, &Born{Name: name})
	return &person, nil
}

// GrowOlder command
func (person *Person) GrowOlder() {
	metaData := make(map[string]interface{})
	metaData["foo"] = "bar"
	aggregate.TrackChangeWithMetadata(person, &AgedOneYear{}, metaData)
}

// Register bind the events to the repository when the aggregate is registered.
func (person *Person) Register(f aggregate.RegisterFunc) {
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

func (person *Person) SerializeSnapshot(aggregate.SnapshotMarshal) ([]byte, error) {
	return json.Marshal(person)
}
func (person *Person) DeserializeSnapshot(f aggregate.SnapshotUnmarshal, d []byte) error {
	return json.Unmarshal(d, person)
}

func TestPersonWithNoEvents(t *testing.T) {
	person := Person{}
	if person.Version() != 0 {
		t.Fatalf("should have version 0 had %d", person.Version())
	}
}

func TestCreateNewPerson(t *testing.T) {
	timeBefore := time.Now().UTC()
	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal("Error when creating person", err.Error())
	}

	if person.Name != "kalle" {
		t.Fatal("Wrong person Name")
	}

	if person.Age != 0 {
		t.Fatal("Wrong person Age")
	}

	if len(person.Events()) != 1 {
		t.Fatal("There should be one event on the person aggregateRoot")
	}

	if person.Version() != 1 {
		t.Fatal("Wrong version on the person aggregateRoot", person.Version())
	}

	if person.Events()[0].Timestamp().Before(timeBefore) {
		t.Fatal("event timestamp before timeBefore")
	}

	if person.Events()[0].Timestamp().After(time.Now().UTC()) {
		t.Fatal("event timestamp after current time")
	}

	if person.Events()[0].GlobalVersion() != 0 {
		t.Fatalf("global version should not be set when event is created, was %d", person.Events()[0].GlobalVersion())
	}

	if !person.UnsavedEvents() {
		t.Fatal("there should be event on the aggregate")
	}
}

func TestCreateNewPersonWithIDFromOutside(t *testing.T) {
	id := "123"
	person, err := CreatePersonWithID(id, "kalle")
	if err != nil {
		t.Fatal("Error when creating person", err.Error())
	}

	if person.ID() != id {
		t.Fatal("Wrong aggregate ID on the person aggregateRoot", person.ID())
	}
}

func TestBlankName(t *testing.T) {
	_, err := CreatePerson("")
	if err == nil {
		t.Fatal("The constructor should return error on blank Name")
	}
}

func TestSetIDOnExistingPerson(t *testing.T) {
	person, err := CreatePerson("Kalle")
	if err != nil {
		t.Fatal("The constructor returned error")
	}

	err = person.SetID("new_id")
	if err == nil {
		t.Fatal("Should not be possible to set ID on already existing person")
	}
}

func TestPersonAgedOneYear(t *testing.T) {
	person, _ := CreatePerson("kalle")
	person.GrowOlder()

	if len(person.Events()) != 2 {
		t.Fatal("There should be two event on the person aggregateRoot", person.Events())
	}

	if person.Events()[len(person.Events())-1].Reason() != "AgedOneYear" {
		t.Fatal("The last event reason should be AgedOneYear", person.Events()[len(person.Events())-1].Reason())
	}

	d, ok := person.Events()[1].Metadata()["foo"]

	if !ok {
		t.Fatal("meta data not present")
	}

	if d.(string) != "bar" {
		t.Fatal("wrong meta data")
	}

	if person.ID() == "" {
		t.Fatal("aggregate ID should not be empty")
	}
}

func TestPersonGrewTenYears(t *testing.T) {
	person, _ := CreatePerson("kalle")
	for i := 1; i <= 10; i++ {
		person.GrowOlder()
	}

	if person.Age != 10 {
		t.Fatal("person has the wrong Age")
	}
}

func TestSetIDFunc(t *testing.T) {
	var counter = 0
	f := func() string {
		counter++
		return fmt.Sprint(counter)
	}

	aggregate.SetIDFunc(f)
	for i := 1; i < 10; i++ {
		person, _ := CreatePerson("kalle")
		if person.ID() != fmt.Sprint(i) {
			t.Fatalf("id not set via the new SetIDFunc, exp: %d got: %s", i, person.ID())
		}
	}
}

func TestIDFuncGeneratingRandomIDs(t *testing.T) {
	var ids = map[string]struct{}{}
	for i := 1; i < 100000; i++ {
		person, _ := CreatePerson("kalle")
		_, exists := ids[person.ID()]
		if exists {
			t.Fatalf("id: %s, already created", person.ID())
		}
		ids[person.ID()] = struct{}{}
	}
}
