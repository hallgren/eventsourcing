package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hallgren/eventsourcing/core"
)

// SQL event store handler
type SQL struct {
	db   *sql.DB
	lock *sync.Mutex
}

// Open connection to database
func Open(db *sql.DB) *SQL {
	return &SQL{
		db: db,
	}
}

// OpenWithSingelWriter prevents multiple writers to save events concurrently
//
// Multiple go routines writing concurrently to sqlite could produce sqlite to lock.
// https://www.sqlite.org/cgi/src/doc/begin-concurrent/doc/begin_concurrent.md
//
// "If there is significant contention for the writer lock, this mechanism can
// be inefficient. In this case it is better for the application to use a mutex
// or some other mechanism that supports blocking to ensure that at most one
// writer is attempting to COMMIT a BEGIN CONCURRENT transaction at a time.
// This is usually easier if all writers are part of the same operating system process."
func OpenWithSingelWriter(db *sql.DB) *SQL {
	return &SQL{
		db:   db,
		lock: &sync.Mutex{},
	}
}

// Close the connection
func (s *SQL) Close() {
	s.db.Close()
}

// Save persists events to the database
func (s *SQL) Save(events []core.Event) error {
	// If no event return no error
	if len(events) == 0 {
		return nil
	}

	if s.lock != nil {
		// prevent multiple writers
		s.lock.Lock()
		defer s.lock.Unlock()
	}
	aggregateID := events[0].AggregateID
	aggregateType := events[0].AggregateType

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return errors.New(fmt.Sprintf("could not start a write transaction, %v", err))
	}
	defer tx.Rollback()

	var currentVersion core.Version
	var version int
	selectStm := `Select version from events where id=$1 and type=$2 order by version desc limit 1`
	err = tx.QueryRow(selectStm, aggregateID, aggregateType).Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return err
	} else if err == sql.ErrNoRows {
		// if no events are saved before set the current version to zero
		currentVersion = core.Version(0)
	} else {
		// set the current version to the last event stored
		currentVersion = core.Version(version)
	}

	// Make sure no other has saved event to the same aggregate concurrently
	if core.Version(currentVersion)+1 != events[0].Version {
		return core.ErrConcurrency
	}

	var lastInsertedID int64
	insert := `Insert into events (id, version, reason, type, timestamp, data, metadata) values ($1, $2, $3, $4, $5, $6, $7)`
	for i, event := range events {
		res, err := tx.Exec(insert, event.AggregateID, event.Version, event.Reason, event.AggregateType, event.Timestamp.Format(time.RFC3339), event.Data, event.Metadata)
		if err != nil {
			return err
		}
		lastInsertedID, err = res.LastInsertId()
		if err != nil {
			return err
		}
		// override the event in the slice exposing the GlobalVersion to the caller
		events[i].GlobalVersion = core.Version(lastInsertedID)
	}
	return tx.Commit()
}

// Get the events from database
func (s *SQL) Get(ctx context.Context, id string, aggregateType string, afterVersion core.Version) (core.Iterator, error) {
	selectStm := `Select seq, id, version, reason, type, timestamp, data, metadata from events where id=$1 and type=$2 and version>$3 order by version asc`
	rows, err := s.db.QueryContext(ctx, selectStm, id, aggregateType, afterVersion)
	if err != nil {
		return nil, err
	}
	return &iterator{rows: rows}, nil
}

// All iterate over all event in GlobalEvents order
func (s *SQL) All(start core.Version, count uint64) (core.Iterator, error) {
	selectStm := `Select seq, id, version, reason, type, timestamp, data, metadata from events where seq >= $1 order by seq asc LIMIT $2`
	rows, err := s.db.Query(selectStm, start, count)
	if err != nil {
		return nil, err
	}
	return &iterator{rows: rows}, nil
}
