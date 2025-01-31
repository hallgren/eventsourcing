> [!NOTE]
> Parts of the repository are currently rewriting and will bring breaking changes. The changes will not affect data stored in event stores, but you have to change how aggregates are constructed and saved. The reason for the change is to simplify the exported API.
>
>  The ongoing work is conducted in this (PR)[https://github.com/hallgren/eventsourcing/pull/147]. Please add comments there.
>
> BR Morgan

# Overview

This set of modules is a post implementation of [@jen20's](https://github.com/jen20) way of implementing event sourcing. You can find the original blog post [here](https://jen20.dev/post/event-sourcing-in-go/) and github repo [here](https://github.com/jen20/go-event-sourcing-sample).

It's structured in two main parts:

* [Event Sourcing](https://github.com/hallgren/eventsourcing?tab=readme-ov-file#event-sourcing) - Model and create events (write side).
* [Projections](https://github.com/hallgren/eventsourcing?tab=readme-ov-file#projections) - Create read-models based on the events (read side).

## Event Sourcing

[Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html) is a technique to make it possible to capture all changes to an application state as a sequence of events.

### Aggregate Root

The *aggregate root* is the central point where events are bound. The aggregate struct needs to embed `eventsourcing.AggreateRoot` to get the aggregate behaviors.

Below, a *Person* aggregate where the Aggregate Root is embedded next to the `Name` and `Age` properties.

```go
type Person struct {
	eventsourcing.AggregateRoot
	Name string
	Age  int
}
```

The aggregate needs to implement the `Transition(event eventsourcing.Event)` and `Register(r eventsourcing.RegisterFunc)` methods to fulfill the aggregate interface. This methods define how events are transformed to build the aggregate state and which events to register into the repository. 

Example of the Transition method from the `Person` aggregate.

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
func (person *Person) Register(r eventsourcing.RegisterFunc) {
    r(&Born{}, &AgedOneYear{})
}
```

The `Born` and `AgedOneYear` events are now registered to the repository when the aggregate is registered.

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

When an aggregate is first created, an event is needed to initialize the state of the aggregate. No event, no aggregate. Below is an example of a constructor that returns the `Person` aggregate and inside it binds an event via the `TrackChange` function. It's possible to define rules that the aggregate must uphold before an event is created, in this case the person's name must not be blank.

```go
// CreatePerson constructor for Person
func CreatePerson(name string) (*Person, error) {
	if name == "" {
		return nil, errors.New("name can't be blank")
	}
	person := Person{}
	person.TrackChange(&person, &Born{Name: name})
	return &person, nil
}
```

When a person is created, more events could be created via methods on the `Person` aggregate. Below is the `GrowOlder` method which in turn triggers the event `AgedOneYear`. This event is tracked on the person aggregate.

```go
// GrowOlder command
func (person *Person) GrowOlder() {
	person.TrackChange(person, &AgedOneYear{})
}
```

Internally the `TrackChange` methods calls the `Transition` method on the aggregate to transform the aggregate based on the newly created event.

To bind metadata to events use the `TrackChangeWithMetadata` method.
  
The internal `Event` looks like this.

```go
type Event struct {
    // aggregate identifier 
    aggregateID string
    // the aggregate version when this event was created
    version         Version
    // the global version is based on all events (this value is only set after the event is saved to the event store) 
    globalVersion   Version
    // aggregate type (Person in the example above)
    aggregateType   string
    // UTC time when the event was created  
    timestamp       time.Time
    // the specific event data specified in the application (Born{}, AgedOneYear{})
    data            interface{}
    // data that don´t belongs to the application state (could be correlation id or other request references)
    metadata        map[string]interface{}
}
```

To access properties on the event you can use the corresponding methods exposing them, e.g `AggregateID()`. This prevent external parties to modify the event from the outside.

### Aggregate ID

The identifier on the aggregate is default set by a random generated string via the crypt/rand pkg. It is possible to change the default behaivior in two ways.

* Set a specific id on the aggregate via the SetID func.

```go
var id = "123"
person := Person{}
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

## Event Repository

The event repository is used to save and retrieve aggregate events. The main functions are:

```go
// saves the events on the aggregate
Save(a aggregate) error

// retrieves and build an aggregate from events based on its identifier
// possible to cancel from the outside
GetWithContext(ctx context.Context, id string, a aggregate) error

// retrieves and build an aggregate from events based on its identifier
Get(id string, a aggregate) error
```

The event repository constructor input values is an event store, this handles the reading and writing of events and builds the aggregate based on the events.

```go
repo := NewRepository(eventStore EventStore) *Repository
```

Here is an example of a person being saved and fetched from the repository.

```go
// the person aggregate has to be registered in the repository
repo.Register(&Person{})

person := person.CreatePerson("Alice")
person.GrowOlder()
repo.Save(person)
twin := Person{}
repo.Get(person.Id, &twin)
```

### Event Store

The only thing an event store handles are events, and it must implement the following interface.

```go
// saves events to the under laying data store.
Save(events []core.Event) error

// fetches events based on identifier and type but also after a specific version. The version is used to load event that happened after a snapshot was taken.
Get(id string, aggregateType string, afterVersion core.Version) (core.Iterator, error)
```

Currently, there are four internal implementations.

* SQL - `go get github.com/hallgren/eventsourcing/eventstore/sql`
* Bolt - `go get github.com/hallgren/eventsourcing/eventstore/bbolt`
* Event Store DB - `go get github.com/hallgren/eventsourcing/eventstore/esdb`
* RAM Memory - part of the main module

And one external.

* [DynamoDB](https://github.com/fd1az/dynamo-es) by [fd1az](https://github.com/fd1az)

### Custom event store

If you want to store events in a database beside the already implemented event stores (`sql`, `bbolt`, `esdb` and `memory`) you can implement, or provide, another event store. It has to implement the `EventStore`  interface to support the eventsourcing.Repository.

```go
type EventStore interface {
    Save(events []core.Event) error
    Get(id string, aggregateType string, afterVersion core.Version) (core.Iterator, error)
}
```

The event store needs to import the `github.com/hallgren/eventsourcing/core` module that expose the `core.Event`, `core.Version` and `core.Iterator` types.

### Encoder

Before an `eventsourcing.Event` is stored into a event store it has to be tranformed into an `core.Event`. This is done with an encoder that serializes the data properties `Data` and `Metadata` into `[]byte`. When a event is fetched the encoder deserialises the `Data` and `Metadata` `[]byte` back into there actual types.

The event repository has a default encoder that uses the `encoding/json` package for serialization/deserialization. It can be replaced by using the `Encoder(e encoder)` method on the event repository and has to follow this interface:

```go
type encoder interface {
	Serialize(v interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}
```

### Event Subscription

The repository expose four possibilities to subscribe to events in realtime as they are saved to the repository.

`All(func (e Event)) *subscription` subscribes to all events.

`AggregateID(func (e Event), events ...aggregate) *subscription` events bound to specific aggregate based on type and identity.
This makes it possible to get events pinpointed to one specific aggregate instance.

`Aggregate(func (e Event), aggregates ...aggregate) *subscription` subscribes to events bound to specific aggregate type. 
 
`Event(func (e Event), events ...interface{}) *subscription` subscribes to specific events. There are no restrictions that the events need
to come from the same aggregate, you can mix and match as you please.

`Name(f func(e Event), aggregate string, events ...string) *subscription` subscribes to events based on aggregate type and event name.

The subscription is realtime and events that are saved before the call to one of the subscribers will not be exposed via the `func(e Event)` function. If the application 
depends on this functionality make sure to call Subscribe() function on the subscriber before storing events in the repository. 

The event subscription enables the application to make use of the reactive patterns and to make it more decoupled. Check out the [Reactive Manifesto](https://www.reactivemanifesto.org/) 
for more detailed information. 

Example on how to set up the event subscription and consume the event `FrequentFlierAccountCreated`

```go
// Setup a memory based repository
repo := eventsourcing.NewRepository(memory.Create())
repo.Register(&FrequentFlierAccountAggregate{})

// subscriber that will trigger on every saved events
s := repo.Subscribers().All(func(e eventsourcing.Event) {
    switch e := event.Data().(type) {
        case *FrequentFlierAccountCreated:
            // e now have type info
            fmt.Println(e)
        }
    }
)

// stop subscription
s.Close()
```

## Snapshot

If an aggregate has a lot of events it can take some time fetching it's event and building the aggregate. This can be optimized with the help of a snapshot. The snapshot is the state of the aggregate on a specific version. Instead of iterating all aggregate events, only the events after the version is iterated and used to build the aggregate. The use of snapshots is optional and is exposed via the snapshot repository.

### Snapshot Repository

The snapshot repository is used to fetch and save aggregate-based snapshots and events (if there are events after the snapshot version). 

The snapshot repository is a layer on top of the event repository.

```go
NewSnapshotRepository(snapshotStore core.SnapshotStore, eventRepo *EventRepository) *SnapshotRepository
```

```go
// fetch the aggregate based on its snapshot and the events after the version of the snapshot
GetWithContext(ctx context.Context, id string, a aggregate) error

// only fetch the aggregate snapshot and not any events
GetSnapshot(ctx context.Context, id string, a aggregate) error

// store the aggregate events and after the snapshot
Save(a aggregate) error

// Store only the aggregate snapshot. Will return an error if there are events that are not stored on the aggregate
SaveSnapshot(a aggregate) error

// expose the underlying event repository.
EventRepository() *EventRepository

// register the aggregate in the underlying event repository
Register(a aggregate)
```

### Snapshot Store

Like the event store's the snapshot repository is built on the same design. The snapshot store has to implement the following methods.

```go
type SnapshotStore interface {
	Save(snapshot Snapshot) error
	Get(ctx context.Context, id, aggregateType string) (Snapshot, error)
}
```

### Unexported aggregate properties

As unexported properties on a struct is not possible to serialize there is the same limitation on aggregates.
To fix this there are optional callback methods that can be added to the aggregate struct.

```go
type SnapshotAggregate interface {
	SerializeSnapshot(SerializeFunc) ([]byte, error)
	DeserializeSnapshot(DeserializeFunc, []byte) error
}
```

Example:

```go
// aggregate
type Person struct {
	eventsourcing.AggregateRoot
	unexported string
}

// snapshot struct
type PersonSnapshot struct {
	UnExported string
}

// callback that maps the aggregate to the snapshot struct with the exported property
func (s *snapshot) SerializeSnapshot(m eventsourcing.SerializeFunc) ([]byte, error) {
	snap := snapshotInternal{
		Unexported: s.unexported,
	}
	return m(snap)
}

// callback to map the snapshot back to the aggregate
func (s *snapshot) DeserializeSnapshot(m eventsourcing.DeserializeFunc, b []byte) error {
	snap := snapshotInternal{}
	err := m(b, &snap)
	if err != nil {
		return err
	}
	s.unexported = snap.UnExported
	return nil
}
```

## Projections

Projections is a way to build read-models based on events. A read-model is way to expose data from events in a different form. Where the form is optimized for read-only queries.

If you want more background on projections check out Derek Comartin projections article [Projections in Event Sourcing: Build ANY model you want!](https://codeopinion.com/projections-in-event-sourcing-build-any-model-you-want/) or Martin Fowler's [CQRS](https://martinfowler.com/bliki/CQRS.html).

### Projection Handler

The Projection handler is the central part where projections are created. It's available from the event repository by the `eventrepo.Projections` property but can also be created standalone.

```go
// access via the event repository
eventRepo := eventsourcing.NewEventRepository(eventstore)
ph := eventRepo.Projections

// standalone without the event repository
ph := eventsourcing.NewProjectionHandler(register, encoder)
```

The projection handler include the event register and a encoder to deserialize events from an event store to application event.

### Projection

A _projection_ is created from the projection handler via the `Projection()` method. The method takes a `fetchFunc` and a `callbackFunc` and returns a pointer to the projection.

```go
p := ph.Projection(f fetchFunc, c callbackFunc)
```

The fetchFunc must return `(core.Iterator, error)`, i.e the same signature that event stores return when they return events.

```go
type fetchFunc func() (core.Iterator, error)
```

The `callbackFunc` is called for every iterated event inside the projection. The event is typed and can be handled in the same way as the aggregate `Transition()` method.

```go
type callbackFunc func(e eventsourcing.Event) error
```

Example: Creates a projection that fetch all events from an event store and handle them in the callbackF.

```go
p := eventRepo.Projections.Projection(es.All(0, 1), func(event eventsourcing.Event) error {
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

RunToEnd fetch events from the event store until it reaches the end of the event stream. A context is passed in making it possible to cancel the projections from the outside.

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

Run will run forever until event consumer is returning an error or if it's canceled from the outside. When it hits the end of the event stream it will start a timer and sleep the time set in the projection property `Pace`.

 ```go
 Run(ctx context.Context, pace time.Duration) error
 ```

A running projection can be triggered manually via `TriggerAsync()` or `TriggerSync()`.

### Projection properties

A projection have a set of properties that can affect it's behaivior.

* **Strict** - Default true and it will trigger an error if a fetched event is not registered in the event `Register`. This force all events to be handled by the callbackFunc.
* **Name** - The name of the projection. Can be useful when debugging multiple running projection. The default name is the index it was created from the projection handler.

### Run multiple projections

#### Group 

A set of projections can run concurrently in a group.

```go
g := ph.Group(p1, p2, p3)
```

A group is started with `g.Start()` where each projection will run in a separate go routine. Errors from a projection can be retrieved from a error channel `g.ErrChan`.

The `g.Stop()` method is used to halt all projections in the group and it returns when all projections has stopped.

```go
// create three projections
p1 := ph.Projection(es.All(0, 1), callbackF)
p2 := ph.Projection(es.All(0, 1), callbackF)
p3 := ph.Projection(es.All(0, 1), callbackF)

// create a group containing the projections
g := ph.Group(p1, p2, p3)

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

The pace of the projection can be changed with the `Pace` property. Default is every 10 second.

If the pace is not fast enough for some senario it's possible to trigger manually.

`TriggerAsync()`: Triggers all projections in the group and return.

`TriggerSync()`: Triggers all projections in the group and wait for them running to the end of there event streams.

#### Race

Compared to a group the race is a one shot operation. Instead of fetching events continuously it's used to iterate and process all existing events and then return.

The `Race()` method starts the projections and run them to the end of there event streams. When all projections are finished the method return.

```go
Race(cancelOnError bool, projections ...*Projection) ([]ProjectionResult, error)
```

If `cancelOnError` is set to true the method will halt all projections and return if any projection is returning an error.

The returned `[]ProjectionResult` is a collection of all projection results.

Race example:

```go
// create two projections
ph := eventsourcing.NewProjectionHandler(register, eventsourcing.EncoderJSON{})
p1 := ph.Projection(es.All(0, 1), callbackF)
p2 := ph.Projection(es.All(0, 1), callbackF)

// true make the race return on error in any projection
result, err := p.Race(true, r1, r2)
```
