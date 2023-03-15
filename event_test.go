package eventsourcing_test

import (
	"testing"

	"github.com/hallgren/eventsourcing"
)

func TestEvent(t *testing.T) {
	e := eventsourcing.Event[PersonEvent]{
		Data: &Born{},
	}
	if e.Reason() != "Born" {
		t.Fatalf("expected Born got %s", e.Reason())
	}
}

func TestDataAs(t *testing.T) {
	type Created struct {
		Name string
		Age  int
	}
	b := Born{Name: "Jonathan"}
	c := Created{}

	e := eventsourcing.Event[PersonEvent]{
		Data: &b,
	}
	err := e.DataAs(&c)
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != c.Name {
		t.Fatal("Name should be the same")
	}
	if c.Age != 0 {
		t.Fatal("Age should be int´s zero value")
	}
}
