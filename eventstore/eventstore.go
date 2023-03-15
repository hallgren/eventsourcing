package eventstore

import (
	"errors"

	"github.com/hallgren/eventsourcing"
)

// ErrEventMultipleAggregates when events holds different id
var ErrEventMultipleAggregates = errors.New("events holds events for more than one aggregate")

// ErrEventMultipleAggregateTypes when events holds different aggregate types
var ErrEventMultipleAggregateTypes = errors.New("events holds events for more than one aggregate type")

// ErrConcurrency when the currently saved version of the aggregate differs from the new ones
var ErrConcurrency = errors.New("concurrency error")

// ErrReasonMissing when the reason is not present in the events
var ErrReasonMissing = errors.New("event holds no reason")

// ValidateEvents make sure the incoming events are valid
func ValidateEvents[T any](aggregateID string, currentVersion eventsourcing.Version, events []eventsourcing.Event[T]) error {
	aggregateType := events[0].AggregateType

	for _, event := range events {
		if event.AggregateID != aggregateID {
			return ErrEventMultipleAggregates
		}

		if event.AggregateType != aggregateType {
			return ErrEventMultipleAggregateTypes
		}

		if currentVersion+1 != event.Version {
			return ErrConcurrency
		}

		if event.Reason() == "" {
			return ErrReasonMissing
		}

		currentVersion = event.Version
	}
	return nil
}

// ValidateEventsNoVersionCheck make sure the incoming events are valid
func ValidateEventsNoVersionCheck[T any](aggregateID string, events []eventsourcing.Event[T]) error {
	aggregateType := events[0].AggregateType
	currentVersion := events[0].Version - 1

	for _, event := range events {
		if event.AggregateID != aggregateID {
			return ErrEventMultipleAggregates
		}

		if event.AggregateType != aggregateType {
			return ErrEventMultipleAggregateTypes
		}

		if currentVersion+1 != event.Version {
			return ErrConcurrency
		}
		if event.Reason() == "" {
			return ErrReasonMissing
		}
		currentVersion = event.Version
	}
	return nil
}
