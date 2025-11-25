package testsuite

import (
	"fmt"
	"testing"

	"github.com/hallgren/eventsourcing/core"
)

func TestFetcherAll(t *testing.T, es core.EventStore, fetcher core.Fetcher) {
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
	defer iter.Close()
	err = verify(iter, 0, len(events))
	if err != nil {
		t.Fatal(err)
	}

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
	err = verify(iter, core.Version(len(events)), len(events2))
	if err != nil {
		t.Fatal(err)
	}
}

func verify(iter core.Iterator, globalVersion core.Version, expCount int) error {
	count := 0
	for iter.Next() {
		count++
		event, err := iter.Value()
		if err != nil {
			return err
		}
		if event.GlobalVersion <= globalVersion {
			return fmt.Errorf("event global version (%q) is lower than previos event %q", event.GlobalVersion, globalVersion)
		}
		globalVersion = event.GlobalVersion
	}
	if count != expCount {
		return fmt.Errorf("expected %d events got %d", expCount, count)
	}
	return nil
}
