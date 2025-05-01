# Overview

This set of modules is a post-implementation of [@jen20's](https://github.com/jen20) approach to implementing event sourcing. You can find the original blog post [here](https://jen20.dev/post/event-sourcing-in-go/) and the GitHub repository [here](https://github.com/jen20/go-event-sourcing-sample).

It is structured into two main parts:

* [Aggregate](https://github.com/hallgren/eventsourcing?tab=readme-ov-file#aggregate) – Modeling and loading/saving aggregates (write side).
* [Consuming events](https://github.com/hallgren/eventsourcing?tab=readme-ov-file#projections) – Handling events and building read models (read side).

## Event Sourcing

[Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html) is a technique that captures all changes to an application’s state as a sequence of events.

### Aggregate

The *aggregate* is the central point where events are bound. The aggregate struct must embed `aggregate.Root` to inherit the required behaviors.

Example of a *Person* aggregate, embedding the Aggregate Root alongside `Name` and `Age` properties:

```go
type Person struct {
	aggregate.Root
	Name string
	Age  int
}
```

The aggregate must implement the `Transition(event eventsourcing.Event)` and `Register(r aggregate.RegisterFunc)` methods to fulfill the aggregate interface. These methods define how events are applied to build the aggregate state and which events are registered in the repository.

Example of the `Transition` method for the `Person` aggregate:

```go
// Transition updates the person's state based on the event
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

The `Born` event sets the `Age` and `Name` properties, and `AgedOneYear` increments the `Age`. This makes the aggregate's state flexible and easy to evolve.

Example of the `Register` method:

```go
// Register registers Person events with the repository
func (person *Person) Register(r aggregate.RegisterFunc) {
	r(&Born{}, &AgedOneYear{})
}
```

The `Born` and `AgedOneYear` events are registered in the repository when the aggregate is registered.

### Event

An event is a plain struct with exported fields representing the event's state.

Example of two events for the `Person` aggregate:

```go
// Initial event
type Born struct {
	Name string
}

// Event that occurs annually
type AgedOneYear struct {}
```

When an aggregate is created, an event is required to initialize its state. Without an event, there is no aggregate. Here's a constructor that initializes a `Person` aggregate by tracking a `Born` event. It also enforces that the person's name must not be blank.

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

After a person is created, additional events can be added through methods on the aggregate. For example, `GrowOlder` triggers the `AgedOneYear` event:

```go
// GrowOlder command
func (person *Person) GrowOlder() {
	aggregate.TrackChange(person, &AgedOneYear{})
}
```

Internally, `aggregate.TrackChange` invokes the `Transition` method to update the aggregate with the new event.

To attach metadata to events, use `aggregate.TrackChangeWithMetadata`.

The `Event` interface has the following methods:

```go
type Event struct {
	AggregateID() string              // Aggregate identifier
	Version() Version                 // Aggregate version when the event was created
	GlobalVersion() Version          // Global version (set after saving to the event store)
	AggregateType() string           // Type of aggregate (e.g., Person)
	Timestamp() time.Time            // UTC timestamp of event creation
	Data() interface{}               // Event-specific data (e.g., Born{}, AgedOneYear{})
	Metadata() map[string]interface{} // Additional metadata (e.g., correlation ID)
}
```

### Aggregate ID

By default, the aggregate ID is set using a randomly generated string via the `crypto/rand` package. This behavior can be changed in two ways:

* Set a specific ID using `SetID`:

```go
var id = "123"
person := Person{}
err := person.SetID(id)
```

* Set a custom ID generator using `aggregate.SetIDFunc`:

```go
var counter = 0
f := func() string {
	counter++
	return fmt.Sprint(counter)
}
aggregate.SetIDFunc(f)
```

## Save/Load Aggregate

To persist and retrieve aggregates, use the following functions from the `aggregate` package. `core.EventStore` defines the interface for the underlying storage.

```go
aggregate.Save(es core.EventStore, a aggregate) error
aggregate.Load(ctx context.Context, es core.EventStore, id string, a aggregate) error
```

Each aggregate must implement the `Register` method and be registered with:

```go
aggregate.Register(&Person{})
```

### Event Store

An event store is responsible for saving and fetching events. It must implement the following interface:

```go
Save(events []core.Event) error
Get(id string, aggregateType string, afterVersion core.Version) (core.Iterator, error)
```

Built-in implementations:

* SQL – `go get github.com/hallgren/eventsourcing/eventstore/sql`
* Bolt – `go get github.com/hallgren/eventsourcing/eventstore/bbolt`
* Event Store DB – `go get github.com/hallgren/eventsourcing/eventstore/esdb`
* In-memory – Included in the main module

External event stores:

* [DynamoDB](https://github.com/fd1az/dynamo-es) by [fd1az](https://github.com/fd1az)
* [SQL pgx driver](https://github.com/CentralConcept/go-eventsourcing-pgx/tree/main/eventstore/pgx)

### Custom Event Store

To implement your own event store, provide a type that satisfies the `core.EventStore` interface:

```go
type EventStore interface {
	Save(events []core.Event) error
	Get(id string, aggregateType string, afterVersion core.Version) (core.Iterator, error)
}
```

Make sure to import `github.com/hallgren/eventsourcing/core` to access required types like `core.Event`, `core.Version`, and `core.Iterator`.

### Encoder

Before an `eventsourcing.Event` is stored, it must be transformed into a `core.Event`. This is done by an encoder that serializes the `Data` and `Metadata` fields into `[]byte`.

The default encoder uses `encoding/json`, but you can provide your own using:

```go
eventsourcing.SetEventEncoder(e Encoder)
```

The `Encoder` interface:

```go
type Encoder interface {
	Serialize(v interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}
```

### Realtime Event Subscription

Realtime event subscription has been removed due to dissatisfaction with the current API. Please open an issue if you'd like it reintroduced.

## Snapshot

If an aggregate has many events, loading it can be slow. Snapshots improve performance by capturing the aggregate state at a specific version, allowing the system to only apply newer events.

### Save/Load Snapshot

Snapshots can only be saved when the aggregate has no pending events:

```go
aggregate.SaveSnapshot(ss core.SnapshotStore, s snapshot) error
aggregate.LoadSnapshot(ctx context.Context, ss core.SnapshotStore, id string, s snapshot) error
aggregate.LoadFromSnapshot(ctx context.Context, es core.EventStore, ss core.SnapshotStore, id string, as aggregateSnapshot) error
```

### Snapshot Store

Like event stores, snapshot stores must implement:

```go
type SnapshotStore interface {
	Save(snapshot Snapshot) error
	Get(ctx context.Context, id, aggregateType string) (Snapshot, error)
}
```

Built-in implementations:

* SQL – `go get github.com/hallgren/eventsourcing/snapshotstore/sql`
* In-memory – Included in the main module

External:

* [SQL pgx driver](https://github.com/CentralConcept/go-eventsourcing-pgx/tree/main/snapshotstore/pgx)

### Unexported Aggregate Properties

Unexported fields cannot be serialized. To work around this, implement optional callback methods:

```go
type snapshot interface {
	SerializeSnapshot(SerializeFunc) ([]byte, error)
	DeserializeSnapshot(DeserializeFunc, []byte) error
}
```

Example:

```go
type Person struct {
	aggregate.Root
	unexported string
}

type PersonSnapshot struct {
	UnExported string
}

func (s *Person) SerializeSnapshot(m aggregate.SnapshotMarshal) ([]byte, error) {
	snap := PersonSnapshot{UnExported: s.unexported}
	return m(snap)
}

func (s *Person) DeserializeSnapshot(m aggregate.SnapshotUnmarshal, b []byte) error {
	snap := PersonSnapshot{}
	if err := m(b, &snap); err != nil {
		return err
	}
	s.unexported = snap.UnExported
	return nil
}
```

You can also change the snapshot encoder with `eventsourcing.SetSnapshotEncoder(e Encoder)`.

## Projections

Projections are used to build read models from events, optimized for querying. For background, see:

* [Projections in Event Sourcing – Derek Comartin](https://codeopinion.com/projections-in-event-sourcing-build-any-model-you-want/)
* [CQRS – Martin Fowler](https://martinfowler.com/bliki/CQRS.html)

### Projection

A projection is created with `eventsourcing.NewProjection(fetchFunc, callbackFunc)`:

```go
p := eventsourcing.NewProjection(f, c)
```

Where:

```go
type fetchFunc func() (core.Iterator, error)
type callbackFunc func(e eventsourcing.Event) error
```

Example:

```go
p := eventsourcing.NewProjection(es.All(0, 1), func(event eventsourcing.Event) error {
	switch e := event.Data().(type) {
	case *Born:
		// handle the event
	}
	return nil
})
```

### Projection Execution

Three execution modes:

#### RunOnce

Runs the projection once:

```go
RunOnce() (bool, ProjectionResult)
```

#### RunToEnd

Runs until the end of the event stream:

```go
RunToEnd(ctx context.Context) ProjectionResult
```

#### Run

Runs continuously until stopped or errored:

```go
Run(ctx context.Context, pace time.Duration) error
```

You can also manually trigger with `TriggerAsync()` or `TriggerSync()`.

### Projection Properties

* **Strict** – Defaults to true; errors if any fetched event is unregistered.
* **Name** – Name of the projection; helpful for debugging.

### Running Multiple Projections

#### Group

Run multiple projections concurrently:

```go
g := eventsourcing.NewProjectionGroup(p1, p2, p3)
g.Start()
defer g.Stop()

select {
case err := <-g.ErrChan:
	// handle error
case <-doneChan:
	// external termination
}
```
