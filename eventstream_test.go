package eventsourcing_test

import (
	"sync"
	"testing"

	"github.com/hallgren/eventsourcing"
)

type TheEvent interface{ theEvent() }

type AnAggregate struct {
	eventsourcing.AggregateRoot[TheEvent]
}

func (a *AnAggregate) Transition(e eventsourcing.Event[TheEvent]) {}

type AnEvent struct {
	Name string
}

func (a *AnEvent) theEvent() {}

type AnotherAggregate struct {
	eventsourcing.AggregateRoot[TheEvent]
}

func (a *AnotherAggregate) Transition(e eventsourcing.Event[TheEvent]) {}

type AnotherEvent struct{}

func (a *AnotherEvent) theEvent() {}

var event = eventsourcing.Event[TheEvent]{Version: 123, Data: &AnEvent{Name: "123"}, AggregateType: "AnAggregate"}
var otherEvent = eventsourcing.Event[TheEvent]{Version: 456, Data: &AnotherEvent{}, AggregateType: "AnotherAggregate"}

func TestSubAll(t *testing.T) {
	var streamEvent *eventsourcing.Event[TheEvent]
	e := eventsourcing.NewEventStream[TheEvent]()
	f := func(e eventsourcing.Event[TheEvent]) {
		streamEvent = &e
	}
	s := e.All(f)
	defer s.Close()
	e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event})

	if streamEvent == nil {
		t.Fatalf("should have received event")
	}
	if streamEvent.Version != event.Version {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version, event.Version)
	}
}

func TestSubSpecificEvent(t *testing.T) {
	var streamEvent *eventsourcing.Event[TheEvent]
	e := eventsourcing.NewEventStream[TheEvent]()
	f := func(e eventsourcing.Event[TheEvent]) {
		streamEvent = &e
	}

	s := e.Event(f, &AnEvent{})
	defer s.Close()
	e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event})

	if streamEvent == nil {
		t.Fatalf("should have received event")
	}

	if streamEvent.Version != event.Version {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version, event.Version)
	}
}

func TestSubAggregateID(t *testing.T) {
	// setup aggregates with identifiers

	anAggregate := AnAggregate{}
	anAggregate.SetID("123")
	anOtherAggregate := AnotherAggregate{}
	anOtherAggregate.SetID("456")

	var streamEvent *eventsourcing.Event[TheEvent]
	e := eventsourcing.NewEventStream[TheEvent]()
	f := func(e eventsourcing.Event[TheEvent]) {
		streamEvent = &e
	}
	s := e.AggregateID(f, &anAggregate, &anOtherAggregate)
	defer s.Close()
	// update with event from the AnAggregate aggregate
	e.Publish(anAggregate.AggregateRoot, []eventsourcing.Event[TheEvent]{event})
	if streamEvent == nil {
		t.Fatalf("should have received event")
	}
	if streamEvent.Version != event.Version {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version, event.Version)
	}

	// update with event from the AnotherAggregate aggregate
	e.Publish(anOtherAggregate.AggregateRoot, []eventsourcing.Event[TheEvent]{otherEvent})
	if streamEvent.Version != otherEvent.Version {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version, otherEvent.Version)
	}
}

func TestSubAggregate(t *testing.T) {
	var streamEvent *eventsourcing.Event[TheEvent]
	e := eventsourcing.NewEventStream[TheEvent]()
	f := func(e eventsourcing.Event[TheEvent]) {
		streamEvent = &e
	}
	s := e.Aggregate(f, &AnAggregate{}, &AnotherAggregate{})
	defer s.Close()

	// update with event from the AnAggregate aggregate
	e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event})
	if streamEvent == nil {
		t.Fatalf("should have received event")
	}
	if streamEvent.Version != event.Version {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version, event.Version)
	}

	// update with event from the AnotherAggregate aggregate
	e.Publish(AnotherAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{otherEvent})
	if streamEvent.Version != otherEvent.Version {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version, otherEvent.Version)
	}
}

func TestSubSpecificEventMultiplePublish(t *testing.T) {
	var streamEvents []*eventsourcing.Event[TheEvent]
	e := eventsourcing.NewEventStream[TheEvent]()
	f := func(e eventsourcing.Event[TheEvent]) {
		streamEvents = append(streamEvents, &e)
	}

	s := e.Event(f, &AnEvent{}, &AnotherEvent{})
	defer s.Close()
	e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event})
	e.Publish(AnotherAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{otherEvent})

	if streamEvents == nil {
		t.Fatalf("should have received event")
	}

	if len(streamEvents) != 2 {
		t.Fatalf("should have received 2 events")
	}

	switch ev := streamEvents[0].Data.(type) {
	case *AnotherEvent:
		t.Fatalf("expecting AnEvent got %q", ev)
	}

	switch ev := streamEvents[1].Data.(type) {
	case *AnEvent:
		t.Fatalf("expecting OtherEvent got %q", ev)
	}

	type AnEvent2 struct {
		TheEvent

		Name string
	}
	switch ev := streamEvents[0].Data.(type) {
	case *AnEvent2:
		t.Fatalf("expecting AnEvent got %q", ev)
	}

}

func TestUpdateNoneSubscribedEvent(t *testing.T) {
	var streamEvent *eventsourcing.Event[TheEvent] = nil
	e := eventsourcing.NewEventStream[TheEvent]()
	f := func(e eventsourcing.Event[TheEvent]) {
		streamEvent = &e
	}
	s := e.Event(f, &AnotherEvent{})
	defer s.Close()
	e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event})

	if streamEvent != nil {
		t.Fatalf("should not have received event %q", streamEvent)
	}
}

func TestManySubscribers(t *testing.T) {
	streamEvent1 := make([]eventsourcing.Event[TheEvent], 0)
	streamEvent2 := make([]eventsourcing.Event[TheEvent], 0)
	streamEvent3 := make([]eventsourcing.Event[TheEvent], 0)
	streamEvent4 := make([]eventsourcing.Event[TheEvent], 0)
	streamEvent5 := make([]eventsourcing.Event[TheEvent], 0)

	e := eventsourcing.NewEventStream[TheEvent]()
	f1 := func(e eventsourcing.Event[TheEvent]) {
		streamEvent1 = append(streamEvent1, e)
	}
	f2 := func(e eventsourcing.Event[TheEvent]) {
		streamEvent2 = append(streamEvent2, e)
	}
	f3 := func(e eventsourcing.Event[TheEvent]) {
		streamEvent3 = append(streamEvent3, e)
	}
	f4 := func(e eventsourcing.Event[TheEvent]) {
		streamEvent4 = append(streamEvent4, e)
	}
	f5 := func(e eventsourcing.Event[TheEvent]) {
		streamEvent5 = append(streamEvent5, e)
	}

	s := e.Event(f1, &AnotherEvent{})
	defer s.Close()
	s = e.Event(f2, &AnotherEvent{}, &AnEvent{})
	defer s.Close()
	s = e.Event(f3, &AnEvent{})
	defer s.Close()
	s = e.All(f4)
	defer s.Close()
	s = e.Aggregate(f5, &AnAggregate{})
	defer s.Close()

	e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event})

	if len(streamEvent1) != 0 {
		t.Fatalf("stream1 should not have any events")
	}

	if len(streamEvent2) != 1 {
		t.Fatalf("stream2 should have one event")
	}

	if len(streamEvent3) != 1 {
		t.Fatalf("stream3 should have one event")
	}

	if len(streamEvent4) != 1 {
		t.Fatalf("stream4 should have one event")
	}

	if len(streamEvent5) != 1 {
		t.Fatalf("stream5 should have one event")
	}
}

func TestParallelPublish(t *testing.T) {
	streamEvent := make([]eventsourcing.Event[TheEvent], 0)
	e := eventsourcing.NewEventStream[TheEvent]()

	// functions to bind to event subscription
	f1 := func(e eventsourcing.Event[TheEvent]) {
		streamEvent = append(streamEvent, e)
	}
	f2 := func(e eventsourcing.Event[TheEvent]) {
		streamEvent = append(streamEvent, e)
	}
	f3 := func(e eventsourcing.Event[TheEvent]) {
		streamEvent = append(streamEvent, e)
	}

	s := e.Event(f1, &AnEvent{})
	defer s.Close()
	s = e.Event(f2, &AnotherEvent{})
	defer s.Close()
	s = e.All(f3)
	defer s.Close()

	wg := sync.WaitGroup{}
	// concurrently update the event stream
	for i := 1; i < 1000; i++ {
		wg.Add(2)
		go func() {
			e.Publish(AnotherAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{otherEvent, otherEvent})
			wg.Done()
		}()
		go func() {
			e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event, event})
			wg.Done()
		}()
	}
	wg.Wait()

	var lastEvent eventsourcing.Event[TheEvent]
	// check that event comes coupled together in four due to the lock in the event stream that makes sure all registered
	// functions are called together and that is not mixed with other events
	for j, event := range streamEvent {
		if j%4 == 0 {
			lastEvent = event
		} else {
			if lastEvent.Reason() != event.Reason() {
				t.Fatal("same event should come in couple of four")
			}
		}
	}
}

func TestClose(t *testing.T) {
	count := 0
	e := eventsourcing.NewEventStream[TheEvent]()
	f := func(e eventsourcing.Event[TheEvent]) {
		count++
	}
	s1 := e.All(f)
	s2 := e.Event(f, &AnEvent{})
	s3 := e.Aggregate(f, &AnAggregate{})
	s4 := e.AggregateID(f, &AnAggregate{})
	s5 := e.All(f)

	// trigger all 5 subscriptions
	e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event})
	if count != 5 {
		t.Fatalf("should have received 5 event")
	}
	// close all subscriptions
	s1.Close()
	s2.Close()
	s3.Close()
	s4.Close()
	s5.Close()

	// new event should not trigger closed subscriptions
	e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event})
	if count != 5 {
		t.Fatalf("should not have received event after subscriptions are closed")
	}
}

func TestName(t *testing.T) {
	var streamEvent eventsourcing.Event[TheEvent]
	var count int
	e := eventsourcing.NewEventStream[TheEvent]()
	f := func(e eventsourcing.Event[TheEvent]) {
		count++
		streamEvent = e
	}
	// triggered
	s := e.Name(f, "AnAggregate", "AnEvent")
	defer s.Close()
	// not triggered
	s2 := e.Name(f, "AnAggregate", "AnEvent2")
	defer s2.Close()
	// not triggered
	s3 := e.Name(f, "AnAggregate2", "AnEvent")
	defer s3.Close()
	e.Publish(AnAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{event})

	if streamEvent.Version != event.Version {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version, event.Version)
	}
	if streamEvent.Data == nil {
		t.Fatalf("should have received event data")
	}

	streamEvent = eventsourcing.Event[TheEvent]{}
	e.Publish(AnotherAggregate{}.AggregateRoot, []eventsourcing.Event[TheEvent]{otherEvent})
	if streamEvent.Version != 0 {
		t.Fatalf("expected zero value")
	}

	if count != 1 {
		t.Fatalf("expected the event function to be hit once")
	}
}
