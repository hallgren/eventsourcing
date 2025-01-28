package eventsourcing_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/aggregate"
	"github.com/hallgren/eventsourcing/core"
)

type AnAggregate struct {
	aggregate.Root
}

func (a *AnAggregate) Transition(e eventsourcing.Event)  {}
func (a *AnAggregate) Register(e aggregate.RegisterFunc) {}

type AnEvent struct {
	Name string
}

type AnotherAggregate struct {
	aggregate.Root
}

func (a *AnotherAggregate) Transition(e eventsourcing.Event)  {}
func (a *AnotherAggregate) Register(e aggregate.RegisterFunc) {}

type AnotherEvent struct{}

var event = eventsourcing.NewEvent(core.Event{Version: 123, AggregateType: "AnAggregate", Data: eventToByte(&AnEvent{Name: "123"})}, &AnEvent{Name: "123"}, nil)
var otherEvent = eventsourcing.NewEvent(core.Event{Version: 456, Data: eventToByte(&AnotherEvent{}), AggregateType: "AnotherAggregate"}, &AnotherEvent{}, nil)

func eventToByte(i interface{}) []byte {
	b, _ := json.Marshal(i)
	return b
}

func TestSubAll(t *testing.T) {
	var streamEvent *eventsourcing.Event
	//e := eventsourcing.NewEventStream()
	f := func(e eventsourcing.Event) {
		streamEvent = &e
	}
	eventsourcing.RealtimeEventStream.Reset()
	s := eventsourcing.RealtimeEventStream.All(f)
	defer s.Close()
	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{event})

	if streamEvent == nil {
		t.Fatalf("should have received event")
	}
	if streamEvent.Version() != event.Version() {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version(), event.Version())
	}
}

func TestSubSpecificEvent(t *testing.T) {
	var streamEvent *eventsourcing.Event
	f := func(e eventsourcing.Event) {
		streamEvent = &e
	}
	eventsourcing.RealtimeEventStream.Reset()

	s := eventsourcing.RealtimeEventStream.Event(f, &AnEvent{})
	defer s.Close()
	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{event})

	if streamEvent == nil {
		t.Fatalf("should have received event")
	}

	if streamEvent.Version() != event.Version() {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version(), event.Version())
	}
}

func TestSubSpecificEventMultiplePublish(t *testing.T) {
	var streamEvents []*eventsourcing.Event
	f := func(e eventsourcing.Event) {
		streamEvents = append(streamEvents, &e)
	}

	eventsourcing.RealtimeEventStream.Reset()
	s := eventsourcing.RealtimeEventStream.Event(f, &AnEvent{}, &AnotherEvent{})
	defer s.Close()
	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{event})
	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{otherEvent})

	if streamEvents == nil {
		t.Fatalf("should have received event")
	}

	if len(streamEvents) != 2 {
		t.Fatalf("should have received 2 events")
	}

	switch ev := streamEvents[0].Data().(type) {
	case *AnotherEvent:
		t.Fatalf("expecting AnEvent got %q", ev)
	}

	switch ev := streamEvents[1].Data().(type) {
	case *AnEvent:
		t.Fatalf("expecting OtherEvent got %q", ev)
	}

	type AnEvent2 struct {
		Name string
	}
	switch ev := streamEvents[0].Data().(type) {
	case *AnEvent2:
		t.Fatalf("expecting AnEvent got %q", ev)
	}

}

func TestUpdateNoneSubscribedEvent(t *testing.T) {
	var streamEvent *eventsourcing.Event = nil
	eventsourcing.RealtimeEventStream.Reset()
	f := func(e eventsourcing.Event) {
		streamEvent = &e
	}
	s := eventsourcing.RealtimeEventStream.Event(f, &AnotherEvent{})
	defer s.Close()
	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{event})

	if streamEvent != nil {
		t.Fatalf("should not have received event %q", streamEvent)
	}
}

func TestManySubscribers(t *testing.T) {
	streamEvent1 := make([]eventsourcing.Event, 0)
	streamEvent2 := make([]eventsourcing.Event, 0)
	streamEvent3 := make([]eventsourcing.Event, 0)
	streamEvent4 := make([]eventsourcing.Event, 0)

	eventsourcing.RealtimeEventStream.Reset()
	f1 := func(e eventsourcing.Event) {
		streamEvent1 = append(streamEvent1, e)
	}
	f2 := func(e eventsourcing.Event) {
		streamEvent2 = append(streamEvent2, e)
	}
	f3 := func(e eventsourcing.Event) {
		streamEvent3 = append(streamEvent3, e)
	}
	f4 := func(e eventsourcing.Event) {
		streamEvent4 = append(streamEvent4, e)
	}

	s := eventsourcing.RealtimeEventStream.Event(f1, &AnotherEvent{})
	defer s.Close()
	s = eventsourcing.RealtimeEventStream.Event(f2, &AnotherEvent{}, &AnEvent{})
	defer s.Close()
	s = eventsourcing.RealtimeEventStream.Event(f3, &AnEvent{})
	defer s.Close()
	s = eventsourcing.RealtimeEventStream.All(f4)
	defer s.Close()

	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{event})

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
}

func TestParallelPublish(t *testing.T) {
	streamEvent := make([]eventsourcing.Event, 0)

	eventsourcing.RealtimeEventStream.Reset()
	// functions to bind to event subscription
	f1 := func(e eventsourcing.Event) {
		streamEvent = append(streamEvent, e)
	}
	f2 := func(e eventsourcing.Event) {
		streamEvent = append(streamEvent, e)
	}
	f3 := func(e eventsourcing.Event) {
		streamEvent = append(streamEvent, e)
	}

	s := eventsourcing.RealtimeEventStream.Event(f1, &AnEvent{})
	defer s.Close()
	s = eventsourcing.RealtimeEventStream.Event(f2, &AnotherEvent{})
	defer s.Close()
	s = eventsourcing.RealtimeEventStream.All(f3)
	defer s.Close()

	wg := sync.WaitGroup{}
	// concurrently update the event stream
	for i := 1; i < 1000; i++ {
		wg.Add(2)
		go func() {
			eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{otherEvent, otherEvent})
			wg.Done()
		}()
		go func() {
			eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{event, event})
			wg.Done()
		}()
	}
	wg.Wait()

	var lastEvent eventsourcing.Event
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
	f := func(e eventsourcing.Event) {
		count++
	}
	eventsourcing.RealtimeEventStream.Reset()
	s1 := eventsourcing.RealtimeEventStream.All(f)
	s2 := eventsourcing.RealtimeEventStream.Event(f, &AnEvent{})
	s5 := eventsourcing.RealtimeEventStream.All(f)

	// trigger all 3 subscriptions
	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{event})
	if count != 3 {
		t.Fatalf("should have received 3 event got %d", count)
	}
	// close all subscriptions
	s1.Close()
	s2.Close()
	s5.Close()

	// new event should not trigger closed subscriptions
	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{event})
	if count != 3 {
		t.Fatalf("should not have received event after subscriptions are closed")
	}
}

func TestName(t *testing.T) {
	var streamEvent eventsourcing.Event
	var count int
	f := func(e eventsourcing.Event) {
		count++
		streamEvent = e
	}
	eventsourcing.RealtimeEventStream.Reset()
	// triggered
	s := eventsourcing.RealtimeEventStream.Name(f, "AnAggregate", "AnEvent")
	defer s.Close()
	// not triggered
	s2 := eventsourcing.RealtimeEventStream.Name(f, "AnAggregate", "AnEvent2")
	defer s2.Close()
	// not triggered
	s3 := eventsourcing.RealtimeEventStream.Name(f, "AnAggregate2", "AnEvent")
	defer s3.Close()
	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{event})

	if streamEvent.Version() != event.Version() {
		t.Fatalf("wrong info in event got %q expected %q", streamEvent.Version(), event.Version())
	}
	if streamEvent.Data() == nil {
		t.Fatalf("should have received event data")
	}

	streamEvent = eventsourcing.Event{}
	eventsourcing.RealtimeEventStream.Publish([]eventsourcing.Event{otherEvent})
	if streamEvent.Version() != 0 {
		t.Fatalf("expected zero value")
	}

	if count != 1 {
		t.Fatalf("expected the event function to be hit once")
	}
}

func TestCloseSubAfterReset(t *testing.T) {
	f := func(e eventsourcing.Event) {}
	s := eventsourcing.RealtimeEventStream.All(f)
	eventsourcing.RealtimeEventStream.Reset()
	s.Close()
}
