package aggregate

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
)

// Aggregate interface to use the aggregate root specific methods
type aggregate interface {
	root() *Root
	Transition(event eventsourcing.Event)
	Register(eventsourcing.RegisterFunc)
}

// AggregateLoad returns the aggregate based on its events
func AggregateLoad(ctx context.Context, es core.EventStore, id string, a aggregate) error {
	if reflect.ValueOf(a).Kind() != reflect.Ptr {
		return ErrAggregateNeedsToBeAPointer
	}

	root := a.root()

	iterator, err := eventsourcing.GetEvents(ctx, es, id, aggregateType(a), root.Version())
	if err != nil {
		return err
	}
	for iterator.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			event, err := iterator.Value()
			if err != nil {
				return err
			}
			buildFromHistory(a, []eventsourcing.Event{event})
		}
	}
	if root.Version() == 0 {
		return eventsourcing.ErrAggregateNotFound
	}
	return nil
}

// AggregateLoadFromSnapshot fetch the aggregate by first get its snapshot and later append events after the snapshot was stored
// This can speed up the load time of aggregates with many events
func AggregateLoadFromSnapshot(ctx context.Context, es core.EventStore, ss core.SnapshotStore, id string, a aggregate) error {
	err := SnapshotLoad(ctx, ss, id, a)
	if err != nil {
		return err
	}
	return AggregateLoad(ctx, es, id, a)
}

// AggregateSave stores the aggregate events and update the snapshot if snapshotstore is present
func AggregateSave(es core.EventStore, a aggregate) error {
	root := a.root()

	// return as quick as possible when no events to process
	if len(root.aggregateEvents) == 0 {
		return nil
	}

	if !eventsourcing.GlobalRegister.AggregateRegistered(a) {
		return fmt.Errorf("%s %w", aggregateType(a), eventsourcing.ErrAggregateNotRegistered)
	}

	globalVersion, err := eventsourcing.SaveEvents(es, root.Events())
	if err != nil {
		return err
	}
	// update the global version on the aggregate
	root.aggregateGlobalVersion = globalVersion

	// set internal properties and reset the events slice
	lastEvent := root.aggregateEvents[len(root.aggregateEvents)-1]
	root.aggregateVersion = lastEvent.Version()
	root.aggregateEvents = []eventsourcing.Event{}

	return nil
}

// AggregateRegister registers the aggregate and its events
func AggregateRegister(a aggregate) {
	eventsourcing.GlobalRegister.Register(a)
}
