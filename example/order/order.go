package order

import (
	"fmt"

	"github.com/hallgren/eventsourcing"
)

type Status string

const (
	Pending  Status = "pending"
	Complete Status = "complete"
)

// Aggregate

// Order is the aggregate protecting the state
type Order struct {
	eventsourcing.AggregateRoot
	Status      Status
	Total       uint
	Outstanding uint
}

// Transition builds the aggregate state based on the events
func (o *Order) Transition(event eventsourcing.Event) {
	switch e := event.Data().(type) {
	case *Created:
		o.Status = Pending
		o.Total = e.Total
		o.Outstanding = e.Total
	case *DiscountApplied:
		o.Outstanding -= e.Discount
	case *Payment:
		o.Outstanding -= e.Amount
	case *Paid:
		o.Status = Complete
	}
}

// Events

// Register binds the events to eventsouring
func (o *Order) Register(r eventsourcing.RegisterFunc) {
	r(
		&Created{},
		&DiscountApplied{},
		&Payment{},
		&Paid{},
	)
}

// Created when the order was created
type Created struct {
	Total uint
}

// DiscountApplied when a discount was applied
type DiscountApplied struct {
	Discount uint
}

// Payment made on the Total amount on the Order
type Payment struct {
	Amount uint
}

// Paid - the order is fully paid
type Paid struct{}

// Commands

// Create creates the initial order
func Create(amount uint) (*Order, error) {
	if amount > 500 {
		return nil, fmt.Errorf("amount can't be higher than 500")
	}

	o := Order{}
	o.TrackChange(&o, &Created{Total: amount})
	return &o, nil
}

// AddDiscount adds discount to the order
func (o *Order) AddDiscount(amount uint) error {
	if o.Outstanding <= amount {
		return fmt.Errorf("discount is larger or same as order outstanding amount")
	}
	o.TrackChange(o, &DiscountApplied{Discount: amount})
	return nil
}

// Pay creates a payment on the order. If the outstanding amount is zero the order
// is paid.
func (o *Order) Pay(amount uint) error {
	if int(o.Outstanding)-int(amount) < 0 {
		return fmt.Errorf("payment is higher than order total amount")
	}

	o.TrackChange(o, &Payment{Amount: amount})

	if o.Outstanding == 0 {
		o.TrackChange(o, &Paid{})
	}
	return nil
}
