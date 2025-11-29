package testsuite

import (
	"fmt"
	"testing"

	"github.com/hallgren/eventsourcing/core"
)

func TestFetcher(t *testing.T, es core.EventStore, fetcher core.Fetcher) {
	globalVersion := core.Version(0)
	aggregateID := AggregateID()

	events := testEvents(aggregateID)
	err := es.Save(events)
	if err != nil {
		t.Fatal(err)
	}

	iter, err := fetcher()
	if err != nil {
		t.Fatal(err)
	}
	err = verify(iter, globalVersion, events)
	if err != nil {
		iter.Close()
		t.Fatal(err)
	}
	iter.Close()

	// set the globalVersion to the length of the events as not all event stores update the
	// globalVersion on the events after they are saved
	globalVersion = core.Version(len(events))

	events2 := testEventsPartTwo(aggregateID)
	err = es.Save(events2)
	if err != nil {
		t.Fatal(err)
	}

	// run fetcher second time - should not restart from first event
	iter2, err := fetcher()
	if err != nil {
		t.Fatal(err)
	}
	defer iter2.Close()
	err = verify(iter2, globalVersion, events2)
	if err != nil {
		t.Fatal(err)
	}
}

// verifies that the events from the iterator has a globalversion higher than sent in globalVersion and that
// the events are the same from the iterator and sent in event.
func verify(iter core.Iterator, globalVersion core.Version, expEvents []core.Event) error {
	i := 0
	for iter.Next() {
		event, err := iter.Value()
		if err != nil {
			return err
		}
		if event.Version != expEvents[i].Version && event.AggregateID != expEvents[i].AggregateID {
			return fmt.Errorf("missmatch in expected event version %q and actual version %q, expected aggregate id %q actual aggregate id %q", expEvents[i].Version, event.Version, expEvents[i].AggregateID, event.AggregateID)
		}
		if event.GlobalVersion <= globalVersion {
			return fmt.Errorf("event global version (%q) is lower than previos event %q", event.GlobalVersion, globalVersion)
		}
		globalVersion = event.GlobalVersion
		i++
	}
	if i != len(expEvents) {
		return fmt.Errorf("expected %d events got %d", len(expEvents), i)
	}
	return nil
}
