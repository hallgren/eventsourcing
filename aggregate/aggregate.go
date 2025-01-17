package aggregate

import (
	"context"
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

type AggregateRepository struct {
	es core.EventStore
	sr *SnapshotRepository
}

func NewAggregateRepository(es core.EventStore, sr *SnapshotRepository) *AggregateRepository {
	return &AggregateRepository{
		es: es,
		sr: sr,
	}
}

// Get returns the aggregate based on its identifier
func Get(ctx context.Context, es core.EventStore, id string, a aggregate) error {
	if reflect.ValueOf(a).Kind() != reflect.Ptr {
		return ErrAggregateNeedsToBeAPointer
	}

	root := a.root()

	iterator, err := eventsourcing.AggregateEvents(ctx, es, id, aggregateType(a), root.Version())
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
			root.BuildFromHistory(a, []eventsourcing.Event{event})
		}
	}
	if root.Version() == 0 {
		return eventsourcing.ErrAggregateNotFound
	}
	return nil
}

// Save stores the aggregate events and update the snapshot if snapshotstore is present
func Save(es core.EventStore, a aggregate) error {
	root := a.root()

	// return as quick as possible when no events to process
	if len(root.aggregateEvents) == 0 {
		return nil
	}

	globalVersion, err := eventsourcing.Save(es, root.Events())
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

// SaveSnapshot calls the underlaying snapshot repository if present
func (ar *AggregateRepository) SaveSnapshot(a aggregate) error {
	if ar.sr != nil {
		return nil
	}
	return ar.sr.Save(a)
}

// Register registers the aggregate and its events
func Register(a aggregate) {
	eventsourcing.Register(a)
}
