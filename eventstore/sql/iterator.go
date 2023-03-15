package sql

import (
	"database/sql"
	"time"

	"github.com/hallgren/eventsourcing"
)

type iterator[T any] struct {
	rows       *sql.Rows
	serializer eventsourcing.Serializer[T]
}

// Next return the next event
func (i *iterator[T]) Next() (eventsourcing.Event[T], error) {
	var globalVersion eventsourcing.Version
	var eventMetadata map[string]interface{}
	var version eventsourcing.Version
	var id, reason, typ, timestamp string
	var data, metadata string
	if !i.rows.Next() {
		if err := i.rows.Err(); err != nil {
			return eventsourcing.Event[T]{}, err
		}
		return eventsourcing.Event[T]{}, eventsourcing.ErrNoMoreEvents
	}
	if err := i.rows.Scan(&globalVersion, &id, &version, &reason, &typ, &timestamp, &data, &metadata); err != nil {
		return eventsourcing.Event[T]{}, err
	}

	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return eventsourcing.Event[T]{}, err
	}

	f, ok := i.serializer.Type(typ, reason)
	if !ok {
		// if the typ/reason is not register jump over the event
		return i.Next()
	}

	eventData := f()
	err = i.serializer.Unmarshal([]byte(data), &eventData)
	if err != nil {
		return eventsourcing.Event[T]{}, err
	}
	if metadata != "" {
		err = i.serializer.Unmarshal([]byte(metadata), &eventMetadata)
		if err != nil {
			return eventsourcing.Event[T]{}, err
		}
	}

	event := eventsourcing.Event[T]{
		AggregateID:   id,
		Version:       version,
		GlobalVersion: globalVersion,
		AggregateType: typ,
		Timestamp:     t,
		Data:          eventData,
		Metadata:      eventMetadata,
	}
	return event, nil
}

// Close closes the iterator
func (i *iterator[T]) Close() {
	i.rows.Close()
}
