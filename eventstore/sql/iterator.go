package sql

import (
	"database/sql"
	"time"

	"github.com/hallgren/eventsourcing/core"
)

type Iterator struct {
	Rows                 *sql.Rows
	CurrentGlobalVersion core.Version
}

// Next return true if there are more data
func (i *Iterator) Next() bool {
	return i.Rows.Next()
}

// Value return the an event
func (i *Iterator) Value() (core.Event, error) {
	var globalVersion core.Version
	var version core.Version
	var id, reason, typ, timestamp string
	var data, metadata []byte

	if err := i.Rows.Scan(&globalVersion, &id, &version, &reason, &typ, &timestamp, &data, &metadata); err != nil {
		return core.Event{}, err
	}

	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return core.Event{}, err
	}

	event := core.Event{
		AggregateID:   id,
		Version:       version,
		GlobalVersion: globalVersion,
		AggregateType: typ,
		Timestamp:     t,
		Data:          data,
		Metadata:      metadata,
		Reason:        reason,
	}
	i.CurrentGlobalVersion = globalVersion
	return event, nil
}

// Close closes the iterator
func (i *Iterator) Close() {
	i.Rows.Close()
}
