package eventsourcing

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hallgren/eventsourcing/core"
)

// Aggregate interface to use the aggregate root specific methods
type aggregate interface {
	root() *Root
	Transition(event Event)
	Register(RegisterFunc)
}

// Get returns the aggregate based on its identifier
func Get(ctx context.Context, es core.EventStore, id string, a aggregate) error {
	if reflect.ValueOf(a).Kind() != reflect.Ptr {
		return ErrAggregateNeedsToBeAPointer
	}

	root := a.root()

	iterator, err := getEvents(ctx, es, id, aggregateType(a), root.Version())
	if err != nil {
		return err
	}
	for iterator.next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			event, err := iterator.value()
			if err != nil {
				return err
			}
			root.BuildFromHistory(a, []Event{event})
		}
	}
	if root.Version() == 0 {
		return ErrAggregateNotFound
	}
	return nil
}

// GetWithSnapshot fetch the aggregate by first get its snapshot and later append events that was saved after the snapshot was stored
func GetWithSnapshot(ctx context.Context, es core.EventStore, ss core.SnapshotStore, id string, a aggregate) error {
	err := GetSnapshot(ctx, ss, id, a)
	if err != nil {
		return err
	}
	return Get(ctx, es, id, a)
}

// Save stores the aggregate events and update the snapshot if snapshotstore is present
func Save(es core.EventStore, a aggregate) error {
	root := a.root()

	// return as quick as possible when no events to process
	if len(root.aggregateEvents) == 0 {
		return nil
	}

	if !globalRegister.aggregateRegistered(a) {
		return fmt.Errorf("%s %w", aggregateType(a), ErrAggregateNotRegistered)
	}

	globalVersion, err := saveEvents(es, root.Events())
	if err != nil {
		return err
	}
	// update the global version on the aggregate
	root.aggregateGlobalVersion = globalVersion

	// set internal properties and reset the events slice
	lastEvent := root.aggregateEvents[len(root.aggregateEvents)-1]
	root.aggregateVersion = lastEvent.Version()
	root.aggregateEvents = []Event{}

	return nil
}

// Register registers the aggregate and its events
func Register(a aggregate) {
	globalRegister.register(a)
}
