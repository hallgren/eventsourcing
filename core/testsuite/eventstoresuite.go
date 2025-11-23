package testsuite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/hallgren/eventsourcing/core"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func AggregateID() string {
	r := seededRand.Intn(999999999999)
	return fmt.Sprintf("%d", r)
}

type eventstoreFunc = func() (core.EventStore, func(), error)

// Status represents the Red, Silver or Gold tier level of a FrequentFlierAccount
type Status int

const (
	StatusRed    Status = iota
	StatusSilver Status = iota
	StatusGold   Status = iota
)

type FrequentFlierAccountCreated struct {
	AccountId         string
	OpeningMiles      int
	OpeningTierPoints int
}

type StatusMatched struct {
	NewStatus Status
}

type FlightTaken struct {
	MilesAdded      int
	TierPointsAdded int
}

var aggregateType = "FrequentFlierAccount"
var timestamp = time.Now()

func eventToByte(i interface{}) []byte {
	b, _ := json.Marshal(i)
	return b
}

func testEvents(aggregateID string) []core.Event {
	metadata := make(map[string]interface{})
	metadata["test"] = "hello"
	history := []core.Event{
		{AggregateID: aggregateID, Version: 1, AggregateType: aggregateType, Timestamp: timestamp, Reason: "FrequentFlierAccountCreated", Data: eventToByte(&FrequentFlierAccountCreated{AccountId: "1234567", OpeningMiles: 10000, OpeningTierPoints: 0}), Metadata: eventToByte(metadata)},
		{AggregateID: aggregateID, Version: 2, AggregateType: aggregateType, Timestamp: timestamp, Reason: "StatusMatched", Data: eventToByte(&StatusMatched{NewStatus: StatusSilver}), Metadata: eventToByte(metadata)},
		{AggregateID: aggregateID, Version: 3, AggregateType: aggregateType, Timestamp: timestamp, Reason: "FlightTaken", Data: eventToByte(&FlightTaken{MilesAdded: 2525, TierPointsAdded: 5}), Metadata: eventToByte(metadata)},
		{AggregateID: aggregateID, Version: 4, AggregateType: aggregateType, Timestamp: timestamp, Reason: "FlightTaken", Data: eventToByte(&FlightTaken{MilesAdded: 2512, TierPointsAdded: 5}), Metadata: eventToByte(metadata)},
		{AggregateID: aggregateID, Version: 5, AggregateType: aggregateType, Timestamp: timestamp, Reason: "FlightTaken", Data: eventToByte(&FlightTaken{MilesAdded: 5600, TierPointsAdded: 5}), Metadata: eventToByte(metadata)},
		{AggregateID: aggregateID, Version: 6, AggregateType: aggregateType, Timestamp: timestamp, Reason: "FlightTaken", Data: eventToByte(&FlightTaken{MilesAdded: 3000, TierPointsAdded: 3}), Metadata: eventToByte(metadata)},
	}
	return history
}

func testEventsPartTwo(aggregateID string) []core.Event {
	history := []core.Event{
		{AggregateID: aggregateID, Version: 7, AggregateType: aggregateType, Timestamp: timestamp, Reason: "FlightTaken", Data: eventToByte(&FlightTaken{MilesAdded: 5600, TierPointsAdded: 5})},
		{AggregateID: aggregateID, Version: 8, AggregateType: aggregateType, Timestamp: timestamp, Reason: "FlightTaken", Data: eventToByte(&FlightTaken{MilesAdded: 3000, TierPointsAdded: 3})},
	}
	return history
}

func testEventOtherAggregate(aggregateID string) core.Event {
	return core.Event{AggregateID: aggregateID, Version: 1, AggregateType: aggregateType, Timestamp: timestamp, Reason: "FrequentFlierAccountCreated", Data: eventToByte(&FrequentFlierAccountCreated{AccountId: "1234567", OpeningMiles: 10000, OpeningTierPoints: 0})}
}

func Test(t *testing.T, esFunc eventstoreFunc) {
	tests := []struct {
		title string
		run   func(es core.EventStore) error
	}{
		{"should save and get events", saveAndGetEvents},
		{"should get events after version", getEventsAfterVersion},
		{"should not save events in wrong version", saveEventsInWrongVersion},
		{"should save and get event concurrently", saveAndGetEventsConcurrently},
		{"should return error when no events", getErrWhenNoEvents},
		{"should get global event order from save", saveReturnGlobalEventOrder},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			es, closeFunc, err := esFunc()
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

func saveAndGetEvents(es core.EventStore) error {
	aggregateID := AggregateID()
	events := testEvents(aggregateID)
	fetchedEvents := []core.Event{}
	err := es.Save(events)
	if err != nil {
		return err
	}
	iterator, err := es.Get(context.Background(), aggregateID, aggregateType, 0)
	if err != nil {
		return err
	}
	for iterator.Next() {
		event, err := iterator.Value()
		if err != nil {
			return err
		}
		fetchedEvents = append(fetchedEvents, event)
	}
	iterator.Close()
	if len(fetchedEvents) != len(testEvents(aggregateID)) {
		return errors.New("wrong number of events returned")
	}

	if fetchedEvents[0].Version != testEvents(aggregateID)[0].Version {
		return errors.New("wrong events returned")
	}

	// Add more events to the same aggregate event stream
	err = es.Save(testEventsPartTwo(aggregateID))
	if err != nil {
		return err
	}
	fetchedEventsIncludingPartTwo := []core.Event{}
	iterator, err = es.Get(context.Background(), aggregateID, aggregateType, 0)
	if err != nil {
		return err
	}
	for iterator.Next() {
		event, err := iterator.Value()
		if err != nil {
			break
		}
		fetchedEventsIncludingPartTwo = append(fetchedEventsIncludingPartTwo, event)
	}
	iterator.Close()

	if len(fetchedEventsIncludingPartTwo) != len(append(testEvents(aggregateID), testEventsPartTwo(aggregateID)...)) {
		return errors.New("wrong number of events returned")
	}

	if fetchedEventsIncludingPartTwo[0].Version != testEvents(aggregateID)[0].Version {
		return errors.New("wrong event version returned")
	}

	if fetchedEventsIncludingPartTwo[0].AggregateID != testEvents(aggregateID)[0].AggregateID {
		return errors.New("wrong event aggregateID returned")
	}

	if fetchedEventsIncludingPartTwo[0].AggregateType != testEvents(aggregateID)[0].AggregateType {
		return errors.New("wrong event aggregateType returned")
	}

	if fetchedEventsIncludingPartTwo[0].Reason != testEvents(aggregateID)[0].Reason {
		return errors.New("wrong event reason returned")
	}
	return nil
}

func getEventsAfterVersion(es core.EventStore) error {
	var fetchedEvents []core.Event
	aggregateID := AggregateID()
	err := es.Save(testEvents(aggregateID))
	if err != nil {
		return err
	}

	iterator, err := es.Get(context.Background(), aggregateID, aggregateType, 1)
	if err != nil {
		return err
	}

	for iterator.Next() {
		event, err := iterator.Value()
		if err != nil {
			break
		}
		fetchedEvents = append(fetchedEvents, event)
	}
	iterator.Close()
	// Should return one less event
	if len(fetchedEvents) != len(testEvents(aggregateID))-1 {
		return fmt.Errorf("wrong number of events returned exp: %d, got:%d", len(fetchedEvents), len(testEvents(aggregateID))-1)
	}
	// first event version should be 2
	if fetchedEvents[0].Version != 2 {
		return fmt.Errorf("wrong events returned")
	}
	return nil
}

func saveEventsInWrongVersion(es core.EventStore) error {
	aggregateID := AggregateID()
	events := testEventsPartTwo(aggregateID)
	err := es.Save(events)

	if !errors.Is(err, core.ErrConcurrency) {
		return errors.New("should not be able to save events that are out of sync compared to the storage order")
	}
	return nil
}

func saveAndGetEventsConcurrently(es core.EventStore) error {
	wg := sync.WaitGroup{}
	var err error
	aggregateID := AggregateID()

	wg.Add(10)
	for i := 0; i < 10; i++ {
		events := testEvents(fmt.Sprintf("%s-%d", aggregateID, i))
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
			events := make([]core.Event, 0)
			for iterator.Next() {
				event, err := iterator.Value()
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

func getErrWhenNoEvents(es core.EventStore) error {
	aggregateID := AggregateID()
	iterator, err := es.Get(context.Background(), aggregateID, aggregateType, 0)
	if err != nil {
		return err
	}
	defer iterator.Close()
	if iterator.Next() {
		return fmt.Errorf("expect no event when no events are saved")
	}
	return nil
}

func saveReturnGlobalEventOrder(es core.EventStore) error {
	aggregateID := AggregateID()
	aggregateID2 := AggregateID()
	events := testEvents(aggregateID)
	err := es.Save(events)
	if err != nil {
		return err
	}
	if events[len(events)-1].GlobalVersion == 0 {
		return fmt.Errorf("expected global event order > 0 on last event got %d", events[len(events)-1].GlobalVersion)
	}
	events2 := []core.Event{testEventOtherAggregate(aggregateID2)}
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
