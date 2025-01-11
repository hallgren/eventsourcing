package aggregate

import (
	"context"
	"reflect"

	"github.com/hallgren/eventsourcing"
)

// Aggregate interface to use the aggregate root specific methods
type aggregate interface {
	root() *Root
	Transition(event eventsourcing.Event)
	Register(eventsourcing.RegisterFunc)
}

type AggregateRepository struct {
	er *eventsourcing.EventRepository
	sr *SnapshotRepository
}

func NewAggregateRepository(er *eventsourcing.EventRepository, sr *SnapshotRepository) *AggregateRepository {
	return &AggregateRepository{
		er: er,
		sr: sr,
	}
}

// Get returns the aggregate based on its identifier
func (ar *AggregateRepository) Get(ctx context.Context, id string, a aggregate) error {
	if reflect.ValueOf(a).Kind() != reflect.Ptr {
		return ErrAggregateNeedsToBeAPointer
	}

	if ar.sr != nil {
		ar.sr.Get(ctx, id, a)
	}
	root := a.root()

	iterator, err := ar.er.AggregateEvents(ctx, id, aggregateType(a), root.Version())
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
func (ar *AggregateRepository) Save(a aggregate) error {
	root := a.root()

	// return as quick as possible when no events to process
	if len(root.aggregateEvents) == 0 {
		return nil
	}

	globalVersion, err := ar.er.Save(root.Events())
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
func (ar *AggregateRepository) Register(a aggregate) {
	ar.er.Register(a)
}
