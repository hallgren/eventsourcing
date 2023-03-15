[![Build Status](https://travis-ci.org/hallgren/eventsourcing.svg?branch=master)](https://travis-ci.org/hallgren/eventsourcing)
[![Go Report Card](https://goreportcard.com/badge/github.com/hallgren/eventsourcing)](https://goreportcard.com/report/github.com/hallgren/eventsourcing)
[![codecov](https://codecov.io/gh/hallgren/eventsourcing/branch/master/graph/badge.svg)](https://codecov.io/gh/hallgren/eventsourcing)

# Overview

This package is an experiment to try to generialize [@jen20's](https://github.com/jen20) way of implementing event sourcing. You can find the original blog post [here](https://jen20.dev/post/event-sourcing-in-go/) and github repo [here](https://github.com/jen20/go-event-sourcing-sample).

## Event Sourcing

[Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html) is a technique to make it possible to capture all changes to an application state as a sequence of events.

### Aggregate Root

The *aggregate root* is the central point where events are bound. The aggregate struct needs to embed `eventsourcing.AggreateRoot` to get the aggregate behaviors.

Aggregate roots have an event type associated with them. This event type can be a `any` type, including an interface to group a set of concrete structs to events.

```go
type EventType interface {
	EventType()
}
```

Below, a *Person* aggregate where the Aggregate Root is embedded next to the `Name` and `Age` properties.

```go
type Person[T EventType] struct {
	eventsourcing.AggregateRoot[T]
	Name string
	Age  int
}
```

The aggregate needs to implement the `Transition(event eventsourcing.Event)` function to fulfill the aggregate interface. This function define how events are transformed to build the aggregate state.

Example of the Transition function from the `Person` aggregate.

```go
// Transition the person state dependent on the events
func (person *Person[T]) Transition(event eventsourcing.Event[T]) {
        switch e := event.Data.(type) {
        case *Born:
                person.Age = 0
                person.Name = e.Name
        case *AgedOneYear:
                person.Age += 1
        }
}
```

In this example we can see that the `Born` event sets the `Person` property `Age` and `Name`, and that the `AgedOneYear` adds one year to the `Age` property. This makes the state of the aggregate flexible and could easily change in the future if required.

### Aggregate Event

An event is a clean struct with exported properties that contains the state of the event.

Example of two events from the `Person` aggregate.

```go
// Initial event
type Born struct {
        Name string
}

// Event that happens once a year
type AgedOneYear struct {}
```

`EventTypes` can be defined from structs by implementing the above interface:

```go
type (*Born) EventType() {}
type (*AgedOneYear) EventType()
```

When an aggregate is first created, an event is needed to initialize the state of the aggregate. No event, no aggregate. Below is an example of a constructor that returns the `Person` aggregate and inside it binds an event via the `TrackChange` function. It's possible to define rules that the aggregate must uphold before an event is created, in this case the person's name must not be blank.

```go
// CreatePerson constructor for Person
func CreatePerson[T EventType](name string) (*Person[EventType], error) {
	if name == "" {
		return nil, errors.New("name can't be blank")
	}
	person := Person[T]{}
	person.TrackChange(&person, &Born{Name: name})
	return &person, nil
}
```

When a person is created, more events could be created via functions on the `Person` aggregate. Below is the `GrowOlder` function which in turn triggers the event `AgedOneYear`. This event is tracked on the person aggregate.

```go
// GrowOlder command
func (person *Person[T]) GrowOlder() {
	person.TrackChange(person, &AgedOneYear{})
}
```

Internally the `TrackChange` functions calls the `Transition` function on the aggregate to transform the aggregate based on the newly created event.

To bind metadata to events use the `TrackChangeWithMetadata` function.
  

The internal `Event` looks like this.

```go
type Event[T any] struct {
    // aggregate identifier 
    AggregateID string
    // the aggregate version when this event was created
    Version         Version
    // the global version is based on all events (this value is only set after the event is saved to the event store) 
    GlobalVersion   Version
    // aggregate type (Person in the example above)
    AggregateType   string
    // UTC time when the event was created  
    Timestamp       time.Time
    // the specific event data specified in the application (Born{}, AgedOneYear{})
    Data            T
    // data that don´t belongs to the application state (could be correlation id or other request references)
    Metadata        map[string]interface{}
}
```

### Aggregate ID

The identifier on the aggregate is default set by a random generated string via the crypt/rand pkg. It is possible to change the default behaivior in two ways.

* Set a specific id on the aggregate via the SetID func.

```go
var id = "123"
person := Person[EventType]{}
err := person.SetID(id)
```

* Change the id generator via the global eventsourcing.SetIDFunc function.

```go
var counter = 0
f := func() string {
	counter++
	return fmt.Sprint(counter)
}

eventsourcing.SetIDFunc(f)
```

## Repository

The repository is used to save and retrieve aggregates. The main functions are:

```go
// saves the events on the aggregate
Save[T any](aggregate Aggregate[T]) error

// retrieves and build an aggregate from events based on its identifier
// possible to cancel from the outside
GetWithContext[T any](ctx context.Context, id string, aggregate Aggregate[T]) error

// retrieves and build an aggregate from events based on its identifier
Get[T any](id string, aggregate Aggregate[T]) error
```

It is possible to save a snapshot of an aggregate reducing the amount of event needed to be fetched and applied.

```go
// saves the aggregate (an error will be returned if there are unsaved events on the aggregate when doing this operation)
SaveSnapshot[T any](aggregate Aggregate[T]) error
```

The repository constructor input values is an event store and a snapshot store, this handles the reading and writing of events and snapshots. We will dig deeper on the internals below.

```go
NewRepository[T any](eventStore EventStore[T], snapshotStore SnapshotStore[T]) *Repository[T]
```

Here is an example of a person being saved and fetched from the repository.

```go
person := person.CreatePerson[EventType]("Alice")
person.GrowOlder()
repo.Save(person)
twin := Person[EventType]{}
repo.Get(person.Id, &twin)
```

### Event Store

The only thing an event store handles are events, and it must implement the following interface.

```go
// saves events to the under laying data store.
Save[T any](events []eventsourcing.Event[T]) error

// fetches events based on identifier and type but also after a specific version. The version is used to load event that happened after a snapshot was taken.
Get[T any](id string, aggregateType string, afterVersion eventsourcing.Version) (eventsourcing.EventIterator[T], error)
```

Currently, there are three implementations.

* SQL
* Bolt
* Event Store DB
* RAM Memory

Post release v0.0.7 event stores `bbolt`, `sql` and `esdb` are their own submodules.
This reduces the dependency graph of the `github.com/hallgren/eventsourcing` module, as each submodule contains their own dependencies not pollute the main module.
Submodules needs to be fetched separately via go get.

`go get github.com/hallgren/eventsourcing/eventstore/sql`  
`go get github.com/hallgren/eventsourcing/eventstore/bbolt`
`go get github.com/hallgren/eventsourcing/eventstore/esdb`

The memory based event store is part of the main module and does not need to be fetched separately.

### Snapshot Handler and Snapshot Store

A snapshot store save and get aggregate snapshots. A snapshot is a fix state of an aggregate on a specific version. The properties of an aggregate have to be exported for them to be saved in the snapshot.

If you want to keep the properties unexported the aggregate has to implement the Marshal/Unmarshal methods.

```go
Marshal(m eventsourcing.MarshalSnapshotFunc) ([]byte, error)
Unmarshal(m eventsourcing.UnmarshalSnapshotFunc, b []byte) error
```
 
Here is an exampel how the Marshal/Unmarshal methods is used in the snapshot aggregate. Marshal maps its properties to a new internal struct with all its
properties exported. The Unmarshal method unmarshal the internal struct and sets the aggregate properties.

```go
type snapshot[T] struct {
	eventsourcing.AggregateRoot[T]
	unexported string
	Exported   string
}

type snapshotInternal struct {
	UnExported string
	Exported   string
}

func (s *snapshot) Marshal(m eventsourcing.MarshalSnapshotFunc) ([]byte, error) {
	snap := snapshotInternal{
		Unexported: s.unexported,
		Exported:   s.Exported,
	}
	return m(snap)
}

func (s *snapshot) Unmarshal(m eventsourcing.UnmarshalSnapshotFunc, b []byte) error {
	snap := snapshotInternal{}
	err := m(b, &snap)
	if err != nil {
		return err
	}
	s.unexported = snap.UnExported
	s.Exported = snap.Exported
	return nil
}
```


The Snapshot Handler is the top layer that integrates with the repository.

```go
// Save transform an aggregate to a snapshot
Save[T any](a interface{}) error {

// Get fetch a snapshot and reconstruct an aggregate
Get[T any](ctx context.Context, id string, a interface{}) error {
```

A Snapshot store is the actual layer that stores the snapshot.

```go
// get snapshot by identifier
Get(ctx context.Context, id, typ string) (eventsource.Snapshot, error)

// saves snapshot
Save(s eventsourcing.Snapshot) error
```

Currently, there are two implementations of the snapshot store.

* SQL
* RAM Memory

Where the SQL snapshot store is a submodule and can be fetched via `go get github.com/hallgren/eventsourcing/snapshotstore/sql`

## Serializer

To store events and snapshots they have to be serialised into `[]byte`. This is handled differently depending on event
store implementation. The sql event store only marshal the event.Data and event.Metadata properties. (The rest is stored
in separate columns), while the bbolt event store marshal the hole event in its key / value database. The memory based event
store does not use a serializer due to it never serialise events to `[]byte`.

To be open to different storage solution the serializer takes as parameter to its constructor a marshal and unmarshal function,
that follows the declaration from the `"encoding/json"` package.

```go
NewSerializer[T any](marshalF MarshalSnapshotFunc, unmarshalF UnmarshalSnapshotFunc) *Serializer[T]

creating a json based serializer: 
serializer := NewSerializer[T any](json.Marshal, json.Unmarshal)
```

The registered event function is used internally inside the event store to set the correct type info when unmarshalling
event data into the `eventsourcing.Event`.

```go
Register[T any](aggregate Aggregate[T], events []func() T)

Register the aggregate Person and the events Born and AgedOneYear (Makes use of the helper method `Events` from the serializer instance):
serializer.Register[EventType](&Person[EventType]{}, serializer.Events(&Born{}, &AgedOneYear{}))
```

### Event Subscription

The repository expose four possibilities to subscribe to events in realtime as they are saved to the repository.

`All[T any](func (e Event[T])) *subscription[T]` subscribes to all events.

`AggregateID[T any](func (e Event[T]), events ...Aggregate[T]) *subscription[T]` events bound to specific aggregate based on type and identity.
This makes it possible to get events pinpointed to one specific aggregate instance.

`Aggregate[T](func (e Event[T]), aggregates ...Aggregate[T]) *subscription[T]` subscribes to events bound to specific aggregate type. 
 
`Event[T](func (e Event[T]), events ...T) *subscription[T]` subscribes to specific events. There are no restrictions that the events need
to come from the same aggregate, you can mix and match as you please.

`Name[T any](f func(e Event[T]), aggregate string, events ...string) *subscription[T]` subscribes to events based on aggregate type and event name.

The subscription is realtime and events that are saved before the call to one of the subscribers will not be exposed via the `func(e Event)` function. If the application 
depends on this functionality make sure to call Subscribe() function on the subscriber before storing events in the repository. 

The event subscription enables the application to make use of the reactive patterns and to make it more decoupled. Check out the [Reactive Manifesto](https://www.reactivemanifesto.org/) 
for more detailed information. 

Example on how to set up the event subscription and consume the event `FrequentFlierAccountCreated`

```go
// Setup a memory based repository
repo := eventsourcing.NewRepository[FrequentFlierEvent](memory.Create[FrequentFlierEvent](), nil)

// subscriber that will trigger on every saved events
s := repo.Subscribers().All(func(e eventsourcing.Event[FrequentFlierEvent]) {
    switch e := event.Data.(type) {
        case *FrequentFlierAccountCreated:
            // e now have type info
            fmt.Println(e)
        }
    }
)

// stop subscription
s.Close()
```

## Custom made components

Parts of this package may not fulfill your application need, either it can be that the event or snapshot stores uses the wrong database for storage.

#### Event Store

A custom-made event store has to implement the following functions to fulfill the interface in the repository.  

```go
type EventStore[T any] interface {
    Save(events []Event[T]) error
    Get(id string, aggregateType string, afterVersion Version) (EventIterator[T], error)
}
```

#### Snapshot Store

If the snapshot store is the thing you need to change here is the interface you need to uphold.

```go
type SnapshotStore interface {
    Get(ctx context.Context, id string, a interface{}) error
    Save(id string, a interface{}) error
}
```
