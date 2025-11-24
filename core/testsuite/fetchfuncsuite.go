package testsuite

import (
	"testing"

	"github.com/hallgren/eventsourcing/core"
)

func TestFetcherAll(t *testing.T, es core.EventStore, fetchFunc core.Fetcher) {
	aggregateID := AggregateID()
	events := testEvents(aggregateID)
	err := es.Save(events)
	if err != nil {
		t.Fatal(err)
	}
	events2 := testEventOtherAggregate(AggregateID())
	err = es.Save([]core.Event{events2})
	if err != nil {
		t.Fatal(err)
	}
	events3 := testEventsPartTwo(aggregateID)
	err = es.Save(events3)
	if err != nil {
		t.Fatal(err)
	}

	iter, err := fetchFunc()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()

	globalVersion := core.Version(0)
	expCount := len(events) + 1 + len(events3)
	count := 0
	for iter.Next() {
		count++
		event, err := iter.Value()
		if err != nil {
			t.Fatal(err)
		}
		if event.GlobalVersion <= globalVersion {
			t.Fatalf("event global version (%q) is lower than previos event %q", event.GlobalVersion, globalVersion)
		}
		globalVersion = event.GlobalVersion
	}
	if count != expCount {
		t.Fatalf("expected %d events got %d", expCount, count)
	}
}
