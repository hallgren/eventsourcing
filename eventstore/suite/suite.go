package suite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/hallgren/eventsourcing"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func AggregateID() string {
	r := seededRand.Intn(999999999999)
	return fmt.Sprintf("%d", r)
}

type eventstoreFunc[T FrequentFlierEvent] func(ser eventsourcing.Serializer[FrequentFlierEvent]) (eventsourcing.EventStore[FrequentFlierEvent], func(), error)

func Test[T FrequentFlierEvent](t *testing.T, esFunc eventstoreFunc[FrequentFlierEvent]) {
	tests := []struct {
		title string
		run   func(es eventsourcing.EventStore[FrequentFlierEvent]) error
	}{
		{"should save and get events", saveAndGetEvents[T]},
		{"should get events after version", getEventsAfterVersion[T]},
		{"should not save events from different aggregates", saveEventsFromMoreThanOneAggregate[T]},
		{"should not save events from different aggregate types", saveEventsFromMoreThanOneAggregateType[T]},
		{"should not save events in wrong order", saveEventsInWrongOrder[T]},
		{"should not save events in wrong version", saveEventsInWrongVersion[T]},
		{"should not save event with no reason", saveEventsWithEmptyReason[T]},
		{"should save and get event concurrently", saveAndGetEventsConcurrently[T]},
		{"should return error when no events", getErrWhenNoEvents[T]},
		{"should get global event order from save", saveReturnGlobalEventOrder[T]},
	}
	ser := eventsourcing.NewSerializer[FrequentFlierEvent](json.Marshal, json.Unmarshal)

	_ = ser.Register(&FrequentFlierAccount[FrequentFlierEvent]{},
		ser.Events(
			&FrequentFlierAccountCreated{},
			&FlightTaken{},
			&StatusMatched{},
		),
	)

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			es, closeFunc, err := esFunc(*ser)
			if err != nil {
				t.Fatal(err)
			}
			err = test.run(es)
			if err != nil {
				// make use of t.Error instead of t.Fatal to make sure the closeFunc is executed
				t.Error(err)
			}
			closeFunc()
		})
	}
}

// Status represents the Red, Silver or Gold tier level of a FrequentFlierAccount
type Status int

const (
	StatusRed    Status = iota
	StatusSilver Status = iota
	StatusGold   Status = iota
)

type FrequentFlierAccount[T FrequentFlierEvent] struct {
	eventsourcing.AggregateRoot[T]
}

func (f *FrequentFlierAccount[T]) Transition(e eventsourcing.Event[T]) {}

type FrequentFlierEvent interface{ frequentFlier() }

type FrequentFlierAccountCreated struct {
	AccountId         string
	OpeningMiles      int
	OpeningTierPoints int
}

func (f *FrequentFlierAccountCreated) frequentFlier() {}

type StatusMatched struct {
	NewStatus Status
}

func (f *StatusMatched) frequentFlier() {}

type FlightTaken struct {
	MilesAdded      int
	TierPointsAdded int
}

func (f *FlightTaken) frequentFlier() {}

var aggregateType = "FrequentFlierAccount"
var timestamp = time.Now()

func testEventsWithID[T FrequentFlierEvent](aggregateID string) []eventsourcing.Event[FrequentFlierEvent] {
	metadata := make(map[string]interface{})
	metadata["test"] = "hello"
	history := []eventsourcing.Event[FrequentFlierEvent]{
		{AggregateID: aggregateID, Version: 1, AggregateType: aggregateType, Timestamp: timestamp, Data: &FrequentFlierAccountCreated{AccountId: "1234567", OpeningMiles: 10000, OpeningTierPoints: 0}, Metadata: metadata},
		{AggregateID: aggregateID, Version: 2, AggregateType: aggregateType, Timestamp: timestamp, Data: &StatusMatched{NewStatus: StatusSilver}, Metadata: metadata},
		{AggregateID: aggregateID, Version: 3, AggregateType: aggregateType, Timestamp: timestamp, Data: &FlightTaken{MilesAdded: 2525, TierPointsAdded: 5}, Metadata: metadata},
		{AggregateID: aggregateID, Version: 4, AggregateType: aggregateType, Timestamp: timestamp, Data: &FlightTaken{MilesAdded: 2512, TierPointsAdded: 5}, Metadata: metadata},
		{AggregateID: aggregateID, Version: 5, AggregateType: aggregateType, Timestamp: timestamp, Data: &FlightTaken{MilesAdded: 5600, TierPointsAdded: 5}, Metadata: metadata},
		{AggregateID: aggregateID, Version: 6, AggregateType: aggregateType, Timestamp: timestamp, Data: &FlightTaken{MilesAdded: 3000, TierPointsAdded: 3}, Metadata: metadata},
	}
	return history
}

func testEvents[T FrequentFlierEvent](aggregateID string) []eventsourcing.Event[FrequentFlierEvent] {
	return testEventsWithID[T](aggregateID)
}

func testEventsPartTwo[T FrequentFlierEvent](aggregateID string) []eventsourcing.Event[FrequentFlierEvent] {
	history := []eventsourcing.Event[FrequentFlierEvent]{
		{AggregateID: aggregateID, Version: 7, AggregateType: aggregateType, Timestamp: timestamp, Data: &FlightTaken{MilesAdded: 5600, TierPointsAdded: 5}},
		{AggregateID: aggregateID, Version: 8, AggregateType: aggregateType, Timestamp: timestamp, Data: &FlightTaken{MilesAdded: 3000, TierPointsAdded: 3}},
	}
	return history
}

func testEventOtherAggregate[T FrequentFlierEvent](aggregateID string) eventsourcing.Event[FrequentFlierEvent] {
	return eventsourcing.Event[FrequentFlierEvent]{AggregateID: aggregateID, Version: 1, AggregateType: aggregateType, Timestamp: timestamp, Data: &FrequentFlierAccountCreated{AccountId: "1234567", OpeningMiles: 10000, OpeningTierPoints: 0}}
}

func saveAndGetEvents[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	aggregateID := AggregateID()
	events := testEvents[FrequentFlierEvent](aggregateID)
	fetchedEvents := []eventsourcing.Event[FrequentFlierEvent]{}
	err := es.Save(events)
	if err != nil {
		return err
	}
	iterator, err := es.Get(context.Background(), aggregateID, aggregateType, 0)
	if err != nil {
		return err
	}
	for {
		event, err := iterator.Next()
		if errors.Is(err, eventsourcing.ErrNoMoreEvents) {
			break
		}
		if err != nil {
			return err
		}
		fetchedEvents = append(fetchedEvents, event)
	}

	iterator.Close()
	if len(fetchedEvents) != len(testEvents[FrequentFlierEvent](aggregateID)) {
		return errors.New("wrong number of events returned")
	}

	if fetchedEvents[0].Version != testEvents[FrequentFlierEvent](aggregateID)[0].Version {
		return errors.New("wrong events returned")
	}

	// Add more events to the same aggregate event stream
	eventsTwo := testEventsPartTwo[FrequentFlierEvent](aggregateID)
	err = es.Save(eventsTwo)
	if err != nil {
		return err
	}
	fetchedEventsIncludingPartTwo := []eventsourcing.Event[FrequentFlierEvent]{}
	iterator, err = es.Get(context.Background(), aggregateID, aggregateType, 0)
	if err != nil {
		return err
	}
	for {
		event, err := iterator.Next()
		if err != nil {
			break
		}
		fetchedEventsIncludingPartTwo = append(fetchedEventsIncludingPartTwo, event)
	}
	iterator.Close()

	if len(fetchedEventsIncludingPartTwo) != len(append(testEvents[T](aggregateID), testEventsPartTwo[FrequentFlierEvent](aggregateID)...)) {
		return errors.New("wrong number of events returned")
	}

	if fetchedEventsIncludingPartTwo[0].Version != testEvents[FrequentFlierEvent](aggregateID)[0].Version {
		return errors.New("wrong event version returned")
	}

	if fetchedEventsIncludingPartTwo[0].AggregateID != testEvents[FrequentFlierEvent](aggregateID)[0].AggregateID {
		return errors.New("wrong event aggregateID returned")
	}

	if fetchedEventsIncludingPartTwo[0].AggregateType != testEvents[FrequentFlierEvent](aggregateID)[0].AggregateType {
		return errors.New("wrong event aggregateType returned")
	}

	if fetchedEventsIncludingPartTwo[0].Reason() != testEvents[FrequentFlierEvent](aggregateID)[0].Reason() {
		return errors.New("wrong event aggregateType returned")
	}

	if fetchedEventsIncludingPartTwo[0].Metadata["test"] != "hello" {
		return errors.New("wrong event meta data returned")
	}

	data, ok := any(fetchedEventsIncludingPartTwo[0].Data).(*FrequentFlierAccountCreated)
	if !ok {
		return errors.New("wrong type in Data")
	}

	if data.OpeningMiles != 10000 {
		return fmt.Errorf("wrong OpeningMiles %d", data.OpeningMiles)
	}
	return nil
}

func getEventsAfterVersion[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	var fetchedEvents []eventsourcing.Event[FrequentFlierEvent]
	aggregateID := AggregateID()
	err := es.Save(testEvents[FrequentFlierEvent](aggregateID))
	if err != nil {
		return err
	}

	iterator, err := es.Get(context.Background(), aggregateID, aggregateType, 1)
	if err != nil {
		return err
	}

	for {
		event, err := iterator.Next()
		if err != nil {
			break
		}
		fetchedEvents = append(fetchedEvents, event)
	}
	iterator.Close()
	// Should return one less event
	if len(fetchedEvents) != len(testEvents[T](aggregateID))-1 {
		return fmt.Errorf("wrong number of events returned exp: %d, got:%d", len(fetchedEvents), len(testEvents[T](aggregateID))-1)
	}
	// first event version should be 2
	if fetchedEvents[0].Version != 2 {
		return fmt.Errorf("wrong events returned")
	}
	return nil
}

func saveEventsFromMoreThanOneAggregate[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	aggregateID := AggregateID()
	aggregateIDOther := AggregateID()
	invalidEvent := append(testEvents[FrequentFlierEvent](aggregateID), testEventOtherAggregate[FrequentFlierEvent](aggregateIDOther))
	err := es.Save(invalidEvent)
	if err == nil {
		return errors.New("should not be able to save events that belongs to more than one aggregate")
	}
	return nil
}

func saveEventsFromMoreThanOneAggregateType[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	aggregateID := AggregateID()
	events := testEvents[FrequentFlierEvent](aggregateID)
	events[1].AggregateType = "OtherAggregateType"

	err := es.Save(events)
	if err == nil {
		return errors.New("should not be able to save events that belongs to other aggregate type")
	}
	return nil
}

func saveEventsInWrongOrder[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	aggregateID := AggregateID()
	events := append(testEvents[FrequentFlierEvent](aggregateID), testEvents[FrequentFlierEvent](aggregateID)[0])
	err := es.Save(events)
	if err == nil {
		return errors.New("should not be able to save events that are in wrong version order")
	}
	return nil
}

func saveEventsInWrongVersion[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	aggregateID := AggregateID()
	events := testEventsPartTwo[FrequentFlierEvent](aggregateID)
	err := es.Save(events)
	if err == nil {
		return errors.New("should not be able to save events that are out of sync compared to the storage order")
	}
	return nil
}

func saveEventsWithEmptyReason[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	aggregateID := AggregateID()
	events := testEvents[FrequentFlierEvent](aggregateID)
	events[2].Data = nil
	err := es.Save(events)
	if err == nil {
		return errors.New("should not be able to save events with empty reason")
	}
	return nil
}

func saveAndGetEventsConcurrently[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	wg := sync.WaitGroup{}
	var err error
	aggregateID := AggregateID()

	wg.Add(10)
	for i := 0; i < 10; i++ {
		events := testEventsWithID[FrequentFlierEvent](fmt.Sprintf("%s-%d", aggregateID, i))
		go func() {
			e := es.Save(events)
			if e != nil {
				err = e
			}
			wg.Done()
		}()
	}
	if err != nil {
		return err
	}

	wg.Wait()
	wg.Add(10)
	for i := 0; i < 10; i++ {
		eventID := fmt.Sprintf("%s-%d", aggregateID, i)
		go func() {
			defer wg.Done()
			iterator, e := es.Get(context.Background(), eventID, aggregateType, 0)
			if e != nil {
				err = e
				return
			}
			events := make([]eventsourcing.Event[FrequentFlierEvent], 0)
			for {
				event, err := iterator.Next()
				if err != nil {
					break
				}
				events = append(events, event)
			}
			iterator.Close()
			if len(events) != 6 {
				err = fmt.Errorf("wrong number of events fetched, expecting 6 got %d", len(events))
				return
			}
		}()
	}
	if err != nil {
		return err
	}
	wg.Wait()
	return nil
}

func getErrWhenNoEvents[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	aggregateID := AggregateID()
	iterator, err := es.Get(context.Background(), aggregateID, aggregateType, 0)
	if err != nil {
		if err != eventsourcing.ErrNoEvents {
			return err
		}
		return nil
	}
	defer iterator.Close()
	_, err = iterator.Next()
	if !errors.Is(err, eventsourcing.ErrNoMoreEvents) {
		return fmt.Errorf("expect error when no events are saved for aggregate")
	}
	return nil
}

func saveReturnGlobalEventOrder[T FrequentFlierEvent](es eventsourcing.EventStore[FrequentFlierEvent]) error {
	aggregateID := AggregateID()
	aggregateID2 := AggregateID()
	events := testEvents[T](aggregateID)
	err := es.Save(events)
	if err != nil {
		return err
	}
	if events[len(events)-1].GlobalVersion == 0 {
		return fmt.Errorf("expected global event order > 0 on last event got %d", events[len(events)-1].GlobalVersion)
	}
	events2 := []eventsourcing.Event[FrequentFlierEvent]{testEventOtherAggregate[T](aggregateID2)}
	err = es.Save(events2)
	if err != nil {
		return err
	}
	if events2[0].GlobalVersion <= events[len(events)-1].GlobalVersion {
		return fmt.Errorf("expected larger global event order got %d", events2[0].GlobalVersion)
	}
	return nil
}

/* re-activate when esdb eventstore have global event order on each stream
func setGlobalVersionOnSavedEvents(es eventsourcing.EventStore) error {
	events := testEvents()
	err := es.Save(events)
	if err != nil {
		return err
	}
	eventsGet, err := es.Get(events[0].AggregateID, events[0].AggregateType, 0)
	if err != nil {
		return err
	}
	var g eventsourcing.Version
	for _, e := range eventsGet {
		g++
		if e.GlobalVersion != g {
			return fmt.Errorf("expected global version to be in sequens exp: %d, was: %d", g, e.GlobalVersion)
		}
	}
	return nil
}
*/
