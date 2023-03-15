package eventsourcing

import (
	"fmt"
	"reflect"
	"sync"
)

// EventStream struct that handles event subscription
type EventStream[T any] struct {
	// makes sure events are delivered in order and subscriptions are persistent
	lock sync.Mutex

	// holds subscribers of aggregate types events
	aggregateTypes map[string][]*subscription[T]
	// holds subscribers of specific aggregates (type and identifier)
	specificAggregates map[string][]*subscription[T]
	// holds subscribers of specific events
	specificEvents map[reflect.Type][]*subscription[T]
	// holds subscribers of all events
	all []*subscription[T]
	// holds subscribers of aggregate and events by name
	names map[string][]*subscription[T]
}

// subscription holds the event function to be triggered when an event is triggering the subscription,
// it also hols a close function to end the subscription.
// event matches the subscription
type subscription[T any] struct {
	eventF func(e Event[T])
	close  func()
}

// Close stops the subscription
func (s *subscription[T]) Close() {
	s.close()
}

// NewEventStream factory function
func NewEventStream[T any]() *EventStream[T] {
	return &EventStream[T]{
		aggregateTypes:     make(map[string][]*subscription[T]),
		specificAggregates: make(map[string][]*subscription[T]),
		specificEvents:     make(map[reflect.Type][]*subscription[T]),
		all:                make([]*subscription[T], 0),
		names:              make(map[string][]*subscription[T]),
	}
}

// Publish calls the functions that are subscribing to the event stream
func (e *EventStream[T]) Publish(agg AggregateRoot[T], events []Event[T]) {
	// the lock prevent other event updates get mixed with this update
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, event := range events {
		e.allPublisher(event)
		e.specificEventPublisher(event)
		e.aggregateTypePublisher(agg, event)
		e.specificAggregatesPublisher(agg, event)
		e.namePublisher(event)
	}
}

// call functions that has registered for all events
func (e *EventStream[T]) allPublisher(event Event[T]) {
	publish(e.all, event)
}

// call functions that has registered for the specific event
func (e *EventStream[T]) specificEventPublisher(event Event[T]) {
	ref := reflect.TypeOf(event.Data)
	if subs, ok := e.specificEvents[ref]; ok {
		publish(subs, event)
	}
}

// call functions that has registered for the aggregate type events
func (e *EventStream[T]) aggregateTypePublisher(agg AggregateRoot[T], event Event[T]) {
	ref := fmt.Sprintf("%s_%s", agg.path(), event.AggregateType)
	if subs, ok := e.aggregateTypes[ref]; ok {
		publish(subs, event)
	}
}

// call functions that has registered for the aggregate type and ID events
func (e *EventStream[T]) specificAggregatesPublisher(agg AggregateRoot[T], event Event[T]) {
	// ref also include the package name ensuring that Aggregate Types can have the same name.
	ref := fmt.Sprintf("%s_%s_%s", agg.path(), event.AggregateType, agg.ID())
	if subs, ok := e.specificAggregates[ref]; ok {
		publish(subs, event)
	}
}

// call functions that has registered for the aggregate type events
func (e *EventStream[T]) namePublisher(event Event[T]) {
	ref := event.AggregateType + "_" + event.Reason()
	if subs, ok := e.names[ref]; ok {
		publish(subs, event)
	}
}

// All subscribe to all events that is stored in the repository
func (e *EventStream[T]) All(f func(e Event[T])) *subscription[T] {
	s := subscription[T]{
		eventF: f,
	}
	s.close = func() {
		e.lock.Lock()
		defer e.lock.Unlock()
		s.eventF = nil
		e.all = clean(e.all)
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	e.all = append(e.all, &s)
	return &s
}

// AggregateID subscribe to events that belongs to aggregate's based on its type and ID
func (e *EventStream[T]) AggregateID(f func(e Event[T]), aggregates ...Aggregate[T]) *subscription[T] {
	s := subscription[T]{
		eventF: f,
	}
	s.close = func() {
		e.lock.Lock()
		defer e.lock.Unlock()
		s.eventF = nil
		// clean all evenF functions that are nil
		for ref, items := range e.specificAggregates {
			e.specificAggregates[ref] = clean(items)
		}
	}
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, a := range aggregates {
		name := reflect.TypeOf(a).Elem().Name()
		root := a.Root()
		ref := fmt.Sprintf("%s_%s_%s", root.path(), name, root.ID())

		// adds one more function to the aggregate
		e.specificAggregates[ref] = append(e.specificAggregates[ref], &s)
	}
	return &s
}

// Aggregate subscribe to events based on the aggregate type
func (e *EventStream[T]) Aggregate(f func(e Event[T]), aggregates ...Aggregate[T]) *subscription[T] {
	s := subscription[T]{
		eventF: f,
	}
	s.close = func() {
		e.lock.Lock()
		defer e.lock.Unlock()
		s.eventF = nil
		// clean all evenF functions that are nil
		for ref, items := range e.aggregateTypes {
			e.aggregateTypes[ref] = clean(items)
		}
	}
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, a := range aggregates {
		name := reflect.TypeOf(a).Elem().Name()
		root := a.Root()
		ref := fmt.Sprintf("%s_%s", root.path(), name)

		// adds one more function to the aggregate
		e.aggregateTypes[ref] = append(e.aggregateTypes[ref], &s)
	}
	return &s
}

// Event subscribe on specific application defined events based on type referencing.
func (e *EventStream[T]) Event(f func(e Event[T]), events ...T) *subscription[T] {
	s := subscription[T]{
		eventF: f,
	}
	s.close = func() {
		e.lock.Lock()
		defer e.lock.Unlock()
		s.eventF = nil
		// clean all evenF functions that are nil
		for ref, items := range e.specificEvents {
			e.specificEvents[ref] = clean(items)
		}
	}
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, event := range events {
		ref := reflect.TypeOf(event)
		// adds one more property to the event type
		e.specificEvents[ref] = append(e.specificEvents[ref], &s)
	}
	return &s
}

// Name subscribe to aggregate name combined with event names. The Name subscriber makes it possible to subscribe to
// events event if the aggregate and event types are within the current application context.
func (e *EventStream[T]) Name(f func(e Event[T]), aggregate string, events ...string) *subscription[T] {
	s := subscription[T]{
		eventF: f,
	}
	s.close = func() {
		e.lock.Lock()
		defer e.lock.Unlock()
		s.eventF = nil
		// clean all evenF functions that are nil
		for ref, items := range e.names {
			e.names[ref] = clean(items)
		}
	}
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, event := range events {
		ref := aggregate + "_" + event
		e.names[ref] = append(e.names[ref], &s)
	}
	return &s
}

// removes subscriptions with event function equal to nil
func clean[T any](items []*subscription[T]) []*subscription[T] {
	for i, s := range items {
		if s.eventF == nil {
			items = append(items[:i], items[i+1:]...)
		}
	}
	return items
}

// publish event to all subscribers
func publish[T any](items []*subscription[T], e Event[T]) {
	for _, s := range items {
		s.eventF(e)
	}
}
