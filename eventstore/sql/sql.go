package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/eventstore"
)

// SQL event store handler
type SQL[T any] struct {
	db         *sql.DB
	serializer eventsourcing.Serializer[T]
}

// Open connection to database
func Open[T any](db *sql.DB, serializer eventsourcing.Serializer[T]) *SQL[T] {
	return &SQL[T]{
		db:         db,
		serializer: serializer,
	}
}

// Close the connection
func (s *SQL[T]) Close() {
	s.db.Close()
}

// Save persists events to the database
func (s *SQL[T]) Save(events []eventsourcing.Event[T]) error {
	// If no event return no error
	if len(events) == 0 {
		return nil
	}
	aggregateID := events[0].AggregateID
	aggregateType := events[0].AggregateType

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return errors.New(fmt.Sprintf("could not start a write transaction, %v", err))
	}
	defer tx.Rollback()

	var currentVersion eventsourcing.Version
	var version int
	selectStm := `Select version from events where id=? and type=? order by version desc limit 1`
	err = tx.QueryRow(selectStm, aggregateID, aggregateType).Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return err
	} else if err == sql.ErrNoRows {
		// if no events are saved before set the current version to zero
		currentVersion = eventsourcing.Version(0)
	} else {
		// set the current version to the last event stored
		currentVersion = eventsourcing.Version(version)
	}

	//Validate events
	err = eventstore.ValidateEvents(aggregateID, currentVersion, events)
	if err != nil {
		return err
	}

	var lastInsertedID int64
	insert := `Insert into events (id, version, reason, type, timestamp, data, metadata) values ($1, $2, $3, $4, $5, $6, $7)`
	for i, event := range events {
		var e, m []byte

		e, err := s.serializer.Marshal(event.Data)
		if err != nil {
			return err
		}
		if event.Metadata != nil {
			m, err = s.serializer.Marshal(event.Metadata)
			if err != nil {
				return err
			}
		}
		res, err := tx.Exec(insert, event.AggregateID, event.Version, event.Reason(), event.AggregateType, event.Timestamp.Format(time.RFC3339), string(e), string(m))
		if err != nil {
			return err
		}
		lastInsertedID, err = res.LastInsertId()
		if err != nil {
			return err
		}
		// override the event in the slice exposing the GlobalVersion to the caller
		events[i].GlobalVersion = eventsourcing.Version(lastInsertedID)
	}
	return tx.Commit()
}

// Get the events from database
func (s *SQL[T]) Get(ctx context.Context, id string, aggregateType string, afterVersion eventsourcing.Version) (eventsourcing.EventIterator[T], error) {
	selectStm := `Select seq, id, version, reason, type, timestamp, data, metadata from events where id=? and type=? and version>? order by version asc`
	rows, err := s.db.QueryContext(ctx, selectStm, id, aggregateType, afterVersion)
	if err != nil {
		return nil, err
	} else if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	i := iterator[T]{rows: rows, serializer: s.serializer}
	return &i, nil
}

// GlobalEvents return count events in order globally from the start posistion
func (s *SQL[T]) GlobalEvents(start, count uint64) ([]eventsourcing.Event[T], error) {
	selectStm := `Select seq, id, version, reason, type, timestamp, data, metadata from events where seq >= ? order by seq asc LIMIT ?`
	rows, err := s.db.Query(selectStm, start, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.eventsFromRows(rows)
}

func (s *SQL[T]) eventsFromRows(rows *sql.Rows) ([]eventsourcing.Event[T], error) {
	var events []eventsourcing.Event[T]
	for rows.Next() {
		var globalVersion eventsourcing.Version
		var eventMetadata map[string]interface{}
		var version eventsourcing.Version
		var id, reason, typ, timestamp string
		var data, metadata string
		if err := rows.Scan(&globalVersion, &id, &version, &reason, &typ, &timestamp, &data, &metadata); err != nil {
			return nil, err
		}

		t, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil, err
		}

		f, ok := s.serializer.Type(typ, reason)
		if !ok {
			// if the typ/reason is not register jump over the event
			continue
		}

		eventData := f()
		err = s.serializer.Unmarshal([]byte(data), &eventData)
		if err != nil {
			return nil, err
		}
		if metadata != "" {
			err = s.serializer.Unmarshal([]byte(metadata), &eventMetadata)
			if err != nil {
				return nil, err
			}
		}

		events = append(events, eventsourcing.Event[T]{
			AggregateID:   id,
			Version:       version,
			GlobalVersion: globalVersion,
			AggregateType: typ,
			Timestamp:     t,
			Data:          eventData,
			Metadata:      eventMetadata,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
