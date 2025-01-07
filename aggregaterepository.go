package eventsourcing

import (
	"context"
	"reflect"
)

type AggregateRepository struct {
	er *EventRepository
	sr *SnapshotRepository
}

func NewAggregateRepository(er *EventRepository, sr *SnapshotRepository) *AggregateRepository {
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

	events, err := ar.er.AggregateEvents(ctx, id, aggregateType(a), root.Version())
	if err != nil {
		return err
	}
	root.BuildFromHistory(a, events)
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
	root.aggregateEvents = []Event{}

	// if a snapshot repository is present store the aggregate
	if ar.sr != nil {
		ar.sr.Save(a)
	}
	return nil
}
