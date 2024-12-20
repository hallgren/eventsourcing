package order_test

import (
	"testing"

	"github.com/hallgren/eventsourcing/example/order"
)

func TestCreateOrder(t *testing.T) {
	o, err := order.Create(1000)
	if err == nil {
		t.Fatal("expected error due to for high amount")
	}

	o, err = order.Create(100)
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != order.Pending {
		t.Fatalf("expected order status to be pending but was %s", o.Status)
	}
	if o.Total != 100 {
		t.Fatalf("expected order total to be 100 but was %d", o.Total)
	}
}

func TestDiscount(t *testing.T) {
	o, err := order.Create(100)
	if err != nil {
		t.Fatal(err)
	}
	err = o.AddDiscount(40)
	if err != nil {
		t.Fatal(err)
	}
	if o.Outstanding != 100-40 {
		t.Fatalf("expected outstanding to be 60 but was %d", o.Outstanding)
	}

	// add discount above orders total amount
	err = o.AddDiscount(80)
	if err == nil {
		t.Fatal("expected error when adding discount above total amount")
	}

	// orders outstanding amount should not be changed
	if o.Outstanding != 100-40 {
		t.Fatalf("expected outstanding to be 60 but was %d", o.Outstanding)
	}

	if o.Status != order.Pending {
		t.Fatalf("order status should be pending but was %s", o.Status)
	}

}

func TestPaid(t *testing.T) {
	o, err := order.Create(100)
	if err != nil {
		t.Fatal(err)
	}

	err = o.Pay(200)
	if err == nil {
		t.Fatal("should not be abel to pay more than total amount")
	}

	err = o.Pay(10)
	if err != nil {
		t.Fatal(err)
	}

	if o.Outstanding != 100-10 {
		t.Fatalf("expected order outstanding to be 90 but was %d", o.Outstanding)
	}

	err = o.Pay(90)
	if err != nil {
		t.Fatal(err)
	}

	if o.Outstanding != 0 {
		t.Fatalf("expected outstanding to be zero but was %d", o.Outstanding)
	}

	if o.Status != order.Complete {
		t.Fatalf("expexted status to be complete but was %s", o.Status)
	}

}
