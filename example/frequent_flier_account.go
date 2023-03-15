package main

import (
	"fmt"

	"github.com/hallgren/eventsourcing"
)

// FrequentFlierAccount represents the state of an instance of the frequent flier
// account aggregate. It tracks changes on itself in the form of domain events.
type FrequentFlierAccount struct {
	miles      int
	tierPoints int
	status     Status
}

// FrequentFlierAccountAggregate is the struct of frequent flier account aggregate
type FrequentFlierAccountAggregate struct {
	eventsourcing.AggregateRoot[FrequentFlierEvent]
	FrequentFlierAccount
}

// Status represents the Red, Silver or Gold tier level of a FrequentFlierAccount
type Status int

// Status represents the Red, Silver or Gold tier level of a FrequentFlierAccount
const (
	StatusRed    Status = iota
	StatusSilver Status = iota
	StatusGold   Status = iota
)

// go:generate stringer -type=Status

type FrequentFlierEvent interface{ frequentFlier() }

// FrequentFlierAccountCreated is the struct of frequent flier accounts created
type FrequentFlierAccountCreated struct {
	AccountID         string
	OpeningMiles      int
	OpeningTierPoints int
}

func (*FrequentFlierAccountCreated) frequentFlier() {}

// StatusMatched is the struct of start matched
type StatusMatched struct {
	NewStatus Status
}

func (*StatusMatched) frequentFlier() {}

// FlightTaken is the struct of flight taken
type FlightTaken struct {
	MilesAdded      int
	TierPointsAdded int
}

func (*FlightTaken) frequentFlier() {}

// PromotedToGoldStatus promoted to gold status
type PromotedToGoldStatus struct{}

func (*PromotedToGoldStatus) frequentFlier() {}

// CreateFrequentFlierAccount constructor
func CreateFrequentFlierAccount(id string) *FrequentFlierAccountAggregate {
	self := FrequentFlierAccountAggregate{}
	self.TrackChange(&self, &FrequentFlierAccountCreated{OpeningMiles: 0, OpeningTierPoints: 0})
	return &self
}

// NewFrequentFlierAccountFromHistory creates a FrequentFlierAccount given a history
// of the changes which have occurred for that account.
func NewFrequentFlierAccountFromHistory[T FrequentFlierEvent](events []eventsourcing.Event[FrequentFlierEvent]) *FrequentFlierAccountAggregate {
	state := FrequentFlierAccountAggregate{}
	state.BuildFromHistory(&state, events)
	return &state
}

// RecordFlightTaken is used to record the fact that a customer has taken a flight
// which should be attached to this frequent flier account. The number of miles and
// tier points which apply are calculated externally.
//
// If recording this flight takes the account over a status boundary, it will
// automatically upgrade the account to the new status level.
func (f *FrequentFlierAccountAggregate) RecordFlightTaken(miles int, tierPoints int) {
	f.TrackChange(f, &FlightTaken{MilesAdded: miles, TierPointsAdded: tierPoints})

	if f.tierPoints > 10 && f.status != StatusSilver {
		f.TrackChange(f, &StatusMatched{NewStatus: StatusSilver})
	}

	if f.tierPoints > 20 && f.status != StatusGold {
		f.TrackChange(f, &PromotedToGoldStatus{})
	}
}

// Transition implements the pattern match against event types used both as part
// of the fold when loading from history and when tracking an individual change.
func (f *FrequentFlierAccountAggregate) Transition(event eventsourcing.Event[FrequentFlierEvent]) {

	switch e := event.Data.(type) {

	case *FrequentFlierAccountCreated:
		f.miles = e.OpeningMiles
		f.tierPoints = e.OpeningTierPoints
		f.status = StatusRed

	case *StatusMatched:
		f.status = e.NewStatus

	case *FlightTaken:
		f.miles = f.miles + e.MilesAdded
		f.tierPoints = f.tierPoints + e.TierPointsAdded

	case *PromotedToGoldStatus:
		f.status = StatusGold
	}
}

// String implements Stringer for FrequentFlierAccount instances.
func (f FrequentFlierAccountAggregate) String() string {

	var reason string
	var aggregateType string

	if len(f.Events()) > 0 {
		reason = f.Events()[0].Reason()
		aggregateType = f.Events()[0].AggregateType
	} else {
		reason = "No reason"
		aggregateType = "No aggregateType"
	}

	format := `FrequentFlierAccount: %s
	Miles: %d
	TierPoints: %d
	Status: %s
	(First Reason: %s)
	(First AggregateType %s)
	(Pending aggregateEvents: %d)
	(aggregateVersion: %d)
`
	return fmt.Sprintf(format, f.ID(), f.miles, f.tierPoints, f.status, reason, aggregateType, len(f.Events()), f.Version())
}
