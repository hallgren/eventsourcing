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
// Collects the business rules for the Order
//
// Rules
// 1. Multiple discounts is not allowed
// 2. Order amount can't be above 500
// 3. Completed order can't be altered
// 4. If a payment has been made it's not possible to alter the discount.

// Order is the aggregate protecting the state
type Order struct {
	eventsourcing.AggregateRoot
	Status      Status
	Total       uint
	Outstanding uint
	Paid        uint
}

// Transition builds the aggregate state based on the events
func (o *Order) Transition(event eventsourcing.Event) {
	switch e := event.Data().(type) {
	case *Created:
		o.Status = Pending
		o.Total = e.Total
		o.Outstanding = e.Total
		o.Paid = 0
	case *DiscountApplied:
		o.Total -= e.Discount
	case *Withdrawed:
		o.Outstanding -= e.Amount
		o.Paid += e.Amount
	case *Paid:
		o.Status = Complete
	}
}

// Events
// Defines all possible events for the Order aggregate

// Register is a eventsouring helper function that must be defined on
// the aggregate.
func (o *Order) Register(r eventsourcing.RegisterFunc) {
	r(
		&Created{},
		&DiscountApplied{},
		&Withdrawed{},
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

// DiscountRemoved when the discount was removed
type DiscountRemoved struct{}

// Withdrawed an amount from the total
type Withdrawed struct {
	Amount uint
}

// Paid - the order is fully paid
type Paid struct{}

// Commands
// Holds the business logic and protects the aggregate (Order) state.
// Events should only be created via commands.

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
	if o.Status == Complete {
		return fmt.Errorf("can't add discount on completed order")
	}
	if o.Outstanding <= amount {
		return fmt.Errorf("discount is larger or same as order outstanding amount")
	}
	o.TrackChange(o, &DiscountApplied{Discount: amount})
	return nil
}

// Pay creates a payment on the order. If the outstanding amount is zero the order
// is paid.
func (o *Order) Pay(amount uint) error {
	if o.Status == Complete {
		return fmt.Errorf("can't pay on completed order")
	}
	if int(o.Outstanding)-int(amount) < 0 {
		return fmt.Errorf("payment is higher than order total amount")
	}

	o.TrackChange(o, &Withdrawed{Amount: amount})

	if o.Outstanding == 0 {
		o.TrackChange(o, &Paid{})
	}
	return nil
}
