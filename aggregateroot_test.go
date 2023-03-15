package eventsourcing_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hallgren/eventsourcing"
)

// Person aggregate
type Person struct {
	eventsourcing.AggregateRoot[PersonEvent]
	Name string
	Age  int
	Dead int
}

type PersonEvent interface {
	personEvent()
}

// Born event
type Born struct {
	Name string
}

func (*Born) personEvent() {}

// AgedOneYear event
type AgedOneYear struct {
}

func (*AgedOneYear) personEvent() {}

// CreatePerson constructor for the Person
func CreatePerson(name string) (*Person, error) {
	if name == "" {
		return nil, errors.New("name can't be blank")
	}
	person := Person{}
	person.TrackChange(&person, &Born{Name: name})
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
	person.TrackChange(&person, &Born{Name: name})
	return &person, nil
}

// GrowOlder command
func (person *Person) GrowOlder() {
	metaData := make(map[string]interface{})
	metaData["foo"] = "bar"
	person.TrackChangeWithMetadata(person, &AgedOneYear{}, metaData)
}

// Transition the person state dependent on the events
func (person *Person) Transition(event eventsourcing.Event[PersonEvent]) {
	switch e := event.Data.(type) {
	case *Born:
		person.Age = 0
		person.Name = e.Name
	case *AgedOneYear:
		person.Age += 1
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

	if person.Events()[0].Timestamp.Before(timeBefore) {
		t.Fatal("event timestamp before timeBefore")
	}

	if person.Events()[0].Timestamp.After(time.Now().UTC()) {
		t.Fatal("event timestamp after current time")
	}

	if person.Events()[0].GlobalVersion != 0 {
		t.Fatalf("global version should not be set when event is created, was %d", person.Events()[0].GlobalVersion)
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

	d, ok := person.Events()[1].Metadata["foo"]

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

	eventsourcing.SetIDFunc(f)
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

func TestMutateEvents(t *testing.T) {
	var m = "mutated from the outside"
	person, _ := CreatePerson("kalle")

	events := person.Events()
	events[0].AggregateType = m
	if person.Events()[0].AggregateType == m {
		t.Fatal("events should not be mutated from the outside")
	}
}
