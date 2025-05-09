# Overview

This set of modules is a post implementation of [@jen20's](https://github.com/jen20) way of implementing event sourcing. You can find the original blog post [here](https://jen20.dev/post/event-sourcing-in-go/) and github repo [here](https://github.com/jen20/go-event-sourcing-sample).

It's structured in two main parts:

* [Aggregate](https://github.com/hallgren/eventsourcing?tab=readme-ov-file#aggregate) - Model and Load/Save aggregates (write side).
* [Consuming events](https://github.com/hallgren/eventsourcing?tab=readme-ov-file#projections) - Handle events and build read-models (read side).

## Event Sourcing

[Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html) is a technique to make it possible to capture all changes to an application state as a sequence of events.

### Aggregate

The *aggregate* is the central point where events are bound. The aggregate struct needs to embed `aggregate.Root` to get the aggregate behaviors.

*Person* aggregate where the Aggregate Root is embedded next to the `Name` and `Age` properties.

```go
type Person struct {
	aggregate.Root
	Name string
	Age  int
}
```

The aggregate needs to implement the `Transition(event eventsourcing.Event)` and `Register(r aggregate.RegisterFunc)` methods to fulfill the aggregate interface.
These methods define how events are transformed to build the aggregate state and which events to register into the repository.

Example of the Transition method on the `Person` aggregate.

```go
// Transition the person state dependent on the events
func (person *Person) Transition(event eventsourcing.Event) {
    switch e := event.Data().(type) {
    case *Born:
            person.Age = 0
            person.Name = e.Name
    case *AgedOneYear:
            person.Age += 1
    }
}
```

The `Born` event sets the `Person` property `Age` and `Name`, and the `AgedOneYear` adds one year to the `Age` property. This makes the state of the aggregate flexible and could easily change in the future if required.

Example or the Register method:

```go
// Register callback method that register Person events to the repository
func (person *Person) Register(r aggregate.RegisterFunc) {
    r(&Born{}, &AgedOneYear{})
}
```

The `Born` and `AgedOneYear` events are now registered to the repository when the aggregate is registered.

### Event

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

When an aggregate is first created, an event is needed to initialize the state of the aggregate. No event, no aggregate.
Example of a constructor that returns the `Person` aggregate and inside it binds an event via the `aggregate.TrackChange` function.
It's possible to define rules that the aggregate must uphold before an event is created, in this case the person's name must not be blank.

```go
// CreatePerson constructor for Person
func CreatePerson(name string) (*Person, error) {
	if name == "" {
		return nil, errors.New("name can't be blank")
	}
	person := Person{}
	aggregate.TrackChange(&person, &Born{Name: name})
	return &person, nil
}
```

When a person is created, more events could be created via methods on the `Person` aggregate. Below is the `GrowOlder` method which in turn triggers the event `AgedOneYear`.

```go
// GrowOlder command
func (person *Person) GrowOlder() {
	aggregate.TrackChange(person, &AgedOneYear{})
}
```

Internally the `aggregate.TrackChange` function calls the `Transition` method on the aggregate to transform the aggregate based on the newly created event.

To bind metadata to events use the `aggregate.TrackChangeWithMetadata` function.
  
The `Event` has the following behaviours..

```go
type Event struct {
    // aggregate identifier 
    AggregateID() string
    // the aggregate version when this event was created
    Version() Version
    // the global version is based on all events (this value is only set after the event is saved to the event store) 
    GlobalVersion() Version
    // aggregate type (Person in the example above)
    AggregateType() string
    // UTC time when the event was created  
    Timestamp() time.Time
    // the specific event data specified in the application (Born{}, AgedOneYear{})
    Data() interface{}
    // data that don´t belongs to the application state (could be correlation id or other request references)
    Metadata() map[string]interface{}
}
```

### Aggregate ID

The identifier on the aggregate is default set by a random generated string via the crypt/rand pkg. It is possible to change the default behavior in two ways.

* Set a specific id on the aggregate via the SetID func.

```go
var id = "123"
person := Person{}
err := person.SetID(id)
```

* Change the id generator via the global aggregate.SetIDFunc function.

```go
var counter = 0
f := func() string {
	counter++
	return fmt.Sprint(counter)
}

aggregate.SetIDFunc(f)
```

## Save/Load Aggregate

To save and load aggregates there are exported functions on the aggregate package. `core.EventStore` is an interface exposing the actual storage system. More on that in later sections.
The `aggregate` interface is automatically applied on the application defined aggregate when `aggregate.Root` is embedded. 

```go
// Save stores the aggregate events in the supplied event store
aggregate.Save(es core.EventStore, a aggregate) error 

// Load returns the aggregate based on its events
aggregate.Load(ctx context.Context, es core.EventStore, id string, a aggregate) error
```

To be able to save and load aggregates they have to be registered and each aggregate has to implement the `Register` method. On top of that the aggregate itself has to be registered via
the `aggregate.Register` function.

```go
// the person aggregate has to be registered in the repository
aggregate.Register(&Person{})
```

### Event Store

The only thing an event store handles are events, and it must implement the following interface.

```go
// saves events to the underlaying data store.
Save(events []core.Event) error

// fetches events based on identifier and type but also after a specific version. The version is used to load events that happened after a snapshot was taken.
Get(id string, aggregateType string, afterVersion core.Version) (core.Iterator, error)
```

There are four implementations in this repository.

* SQL - `go get github.com/hallgren/eventsourcing/eventstore/sql`
* Bolt - `go get github.com/hallgren/eventsourcing/eventstore/bbolt`
* Event Store DB - `go get github.com/hallgren/eventsourcing/eventstore/esdb`
* Kurrent DB - `go get github.com/hallgren/eventsourcing/eventstore/kurrent`
* RAM Memory - part of the main module

External event stores:

* [DynamoDB](https://github.com/fd1az/dynamo-es) by [fd1az](https://github.com/fd1az)
* [SQL pgx driver](https://github.com/CentralConcept/go-eventsourcing-pgx/tree/main/eventstore/pgx)

### Custom event store

If you want to store events in a database beside the already implemented event stores you can implement, or provide, another event store. It has to implement the `core.EventStore` 
interface to support the eventsourcing.EventRepository.

```go
type EventStore interface {
    Save(events []core.Event) error
    Get(id string, aggregateType string, afterVersion core.Version) (core.Iterator, error)
}
```

The event store needs to import the `github.com/hallgren/eventsourcing/core` module that expose the `core.Event`, `core.Version` and `core.Iterator` types.

### Encoder

Before an `eventsourcing.Event` is stored into a event store it has to be transformed into an `core.Event`. This is done with an encoder that serializes the data properties `Data` and `Metadata` into `[]byte`.
When an event is fetched the encoder deserialize the `Data` and `Metadata` `[]byte` back into their actual types.

The default encoder uses the `encoding/json` package for serialization/deserialization. It can be replaced by using the `eventsourcing.SetEventEncoder(e Encoder)` function on the eventsourcing  package, it has to follow this interface:

```go
type Encoder interface {
	Serialize(v interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}
```

### Realtime Event Subscription

For now the real time event subscription has been removed as I'm not satisfied with the exported API. Please fill an issue if you want it back.

## Snapshot

If an aggregate has a lot of events it can take some time fetching and building the aggregate. This can be optimized with the help of a snapshot.
The snapshot is the state of the aggregate on a specific version. Instead of iterating all events, only the events after the version are iterated and
used to build the aggregate. The use of snapshots is optional and is exposed via the snapshot functions on the aggregate package.

### Save/Load Snapshot 

It's only possible to save a snapshot if it has no pending events, meaning that the aggregate events are saved before saving the snapshot.

```go
// Saves a snapshot
aggregate.SaveSnapshot(ss core.SnapshotStore, s snapshot) error

// Loads the aggregate only from the snapshot state not adding events that were saved after the snapshot was taken
aggregate.LoadSnapshot(ctx context.Context, ss core.SnapshotStore, id string, s snapshot) error

// Loads the aggregate from the snapshot and also adds events
aggregate.LoadFromSnapshot(ctx context.Context, es core.EventStore, ss core.SnapshotStore, id string, as aggregateSnapshot) error
```

### Snapshot Store

Like the event store's the snapshot repository is built on the same design. The snapshot store has to implement the following methods.

```go
type SnapshotStore interface {
	Save(snapshot Snapshot) error
	Get(ctx context.Context, id, aggregateType string) (Snapshot, error)
}
```

There are two implementations in this repository.

* SQL - `go get github.com/hallgren/eventsourcing/snapshotstore/sql`
* RAM Memory - part of the main module

External event stores:

* [SQL pgx driver](https://github.com/CentralConcept/go-eventsourcing-pgx/tree/main/snapshotstore/pgx)

### Unexported aggregate properties

As unexported properties on a struct are not possible to serialize there is the same limitation on aggregates.
To fix this there are optional callback methods that can be added to the aggregate struct.

```go
type snapshot interface {
	SerializeSnapshot(SerializeFunc) ([]byte, error)
	DeserializeSnapshot(DeserializeFunc, []byte) error
}
```

Example:

```go
// aggregate
type Person struct {
	aggregate.Root
	unexported string
}

// snapshot struct
type PersonSnapshot struct {
	UnExported string
}

// callback that maps the aggregate to the snapshot struct with the exported property
func (s *Person) SerializeSnapshot(m aggregate.SnapshotMarshal) ([]byte, error) {
	snap := PersonSnapshot{
		Unexported: s.unexported,
	}
	return m(snap)
}

// callback to map the snapshot back to the aggregate
func (s *Person) DeserializeSnapshot(m aggregate.SnapshotUnmarshal, b []byte) error {
	snap := PersonSnapshot{}
	err := m(b, &snap)
	if err != nil {
		return err
	}
	s.unexported = snap.UnExported
	return nil
}
```

It's possible to change the default json encoder by the `eventsourcing.SetSnapshotEncoder(e Encoder)` function.

## Projections

Projections is a way to build read-models based on events. A read-model is a way to expose data from events in a different form. Where the form is optimized for read-only queries.

If you want more background on projections check out Derek Comartin projections article [Projections in Event Sourcing: Build ANY model you want!](https://codeopinion.com/projections-in-event-sourcing-build-any-model-you-want/) or Martin Fowler's [CQRS](https://martinfowler.com/bliki/CQRS.html).

### Projection

A _projection_ is created from the `eventsourcing.NewProjection` function. The method takes a `fetchFunc` and a `callbackFunc` and returns a pointer to the projection.

```go
p := pr.Projection(f fetchFunc, c callbackFunc)
```

The fetchFunc must return `(core.Iterator, error)`, i.e the same signature that event stores return when they return events.

```go
type fetchFunc func() (core.Iterator, error)
```

The `callbackFunc` is called for every iterated event inside the projection. The event is typed and can be handled in the same way as the aggregate `Transition()` method.

```go
type callbackFunc func(e eventsourcing.Event) error
```

Example: Creates a projection that fetches all events from an event store and handle them in the callbackF.

```go
p := eventsourcing.NewProjection(es.All(0, 1), func(event eventsourcing.Event) error {
	switch e := event.Data().(type) {
	case *Born:
		// handle the event
	}
	return nil
})
```

### Projection execution

A projection can be started in three different ways.

#### RunOnce

RunOnce fetch events from the event store one time. It returns true if there were events to iterate otherwise false.

```go
RunOnce() (bool, ProjectionResult)
```

#### RunToEnd

RunToEnd fetches events from the event store until it reaches the end of the event stream. A context is passed in making it possible to cancel the projections from the outside.

```go
RunToEnd(ctx context.Context) ProjectionResult
```

`RunOnce` and `RunToEnd` both return a ProjectionResult 

```go
type ProjectionResult struct {
	Error          		error
	ProjectionName 		string
	LastHandledEvent	Event
}
```

* **Error** Is set if the projection returned an error
* **ProjectionName** Is the name of the projection
* **LastHandledEvent** The last successfully handled event (can be useful during debugging)

#### Run

Run will run forever until the event consumer is returning an error or if it's canceled from the outside. When it hits the end of the event stream it will start a timer and sleep the time set in the projection property `Pace`.

 ```go
 Run(ctx context.Context, pace time.Duration) error
 ```

A running projection can be triggered manually via `TriggerAsync()` or `TriggerSync()`.

### Projection properties

A projection has a set of properties that can affect its behavior.

* **Strict** - Default true and it will trigger an error if a fetched event is not registered in the event `Register`. This forces all events to be handled by the callbackFunc.
* **Name** - The name of the projection. Can be useful when debugging multiple running projections. The default name is the index it was created from the projection handler.

### Run multiple projections

#### Group 

A set of projections can run concurrently in a group.

```go
g := eventsourcing.NewProjectionGroup(p1, p2, p3)
```

A group is started with `g.Start()` where each projection will run in a separate go routine. Errors from a projection can be retrieved from an error channel `g.ErrChan`.

The `g.Stop()` method is used to halt all projections in the group and it returns when all projections have stopped.

```go
// create three projections
p1 := eventsourcing.NewProjection(es.All(0, 1), callbackF)
p2 := eventsourcing.NewProjection(es.All(0, 1), callbackF)
p3 := eventsourcing.NewProjection(es.All(0, 1), callbackF)

// create a group containing the projections
g := eventsourcing.NewProjectionGroup(p1, p2, p3)

// Start runs all projections concurrently
g.Start()

// Stop terminate all projections and wait for them to return
defer g.Stop()

// handling error in projection or termination from outside
select {
	case err := <-g.ErrChan:
		// handle the error
	case <-doneChan:
		// stop signal from the out side
		return
}
```

The pace of the projection can be changed with the `Pace` property. Default is every 10 seconds.

If the pace is not fast enough for some scenario it's possible to trigger manually.

`TriggerAsync()`: Triggers all projections in the group and returns.

`TriggerSync()`: Triggers all projections in the group and waits for them running to the end of their event streams.

#### Race

Compared to a group the race is a one shot operation. Instead of fetching events continuously it's used to iterate and process all existing events and then return.

The `Race()` method starts the projections and runs them to the end of their event streams. When all projections are finished the method returns.

```go
eventsourcing.ProjectionsRace(cancelOnError bool, projections ...*Projection) ([]ProjectionResult, error)
```

If `cancelOnError` is set to true the method will halt all projections and return if any projection is returning an error.

The returned `[]ProjectionResult` is a collection of all projection results.

Race example:

```go
// create two projections
p1 := eventsourcing.NewProjection(es.All(0, 1), callbackF)
p2 := eventsourcing.NewProjection(es.All(0, 1), callbackF)

// true make the race return on error in any projection
result, err := eventsourcing.ProjectionsRace(true, r1, r2)
```
