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

var postgresStm = []string{`CREATE TABLE IF NOT EXISTS events (
	seq SERIAL PRIMARY KEY,
	id VARCHAR NOT NULL,
	version INTEGER,
	reason VARCHAR,
	type VARCHAR,
	timestamp VARCHAR,
	data BYTEA,
	metadata BYTEA,
	UNIQUE (id, type, version)
);`,
	`CREATE INDEX IF NOT EXISTS id_type ON events (id, type);`,
}

// Postgres event store handler
type Postgres struct {
	db   *sql.DB
	lock *sync.Mutex
}

// NewPostgres connection to database
func NewPostgres(db *sql.DB) (*Postgres, error) {
	// make sure the schema is migrated
	if err := migrate(db, postgresStm); err != nil {
		return nil, err
	}
	s := &Postgres{
		db: db,
	}
	return s, nil
}

// Close the connection
func (s *Postgres) Close() {
	s.db.Close()
}

// Save persists events to the database
func (s *Postgres) Save(events []core.Event) error {
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
	selectStm := `SELECT version FROM events WHERE id=$1 and type=$2 ORDER BY version DESC LIMIT 1`
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
	insert := `INSERT INTO events (id, version, reason, type, timestamp, data, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING seq`
	for i, event := range events {
		err := tx.QueryRow(insert, event.AggregateID, event.Version, event.Reason, event.AggregateType, event.Timestamp.Format(time.RFC3339), event.Data, event.Metadata).Scan(&lastInsertedID)
		if err != nil {
			return err
		}
		// override the event in the slice exposing the GlobalVersion to the caller
		events[i].GlobalVersion = core.Version(lastInsertedID)
	}
	return tx.Commit()
}

// Get the events from database
func (s *Postgres) Get(ctx context.Context, id string, aggregateType string, afterVersion core.Version) (core.Iterator, error) {
	selectStm := `SELECT seq, id, version, reason, type, timestamp, data, metadata FROM events WHERE id=$1 AND type=$2 AND version>$3 ORDER BY version ASC`
	rows, err := s.db.QueryContext(ctx, selectStm, id, aggregateType, afterVersion)
	if err != nil {
		return nil, err
	}
	return &iterator{rows: rows}, nil
}

// All iterate over all event in GlobalEvents order
func (s *Postgres) All(start core.Version, count uint64) func() (core.Iterator, error) {
	return func() (core.Iterator, error) {
		selectStm := `SELECT seq, id, version, reason, type, timestamp, data, metadata FROM events WHERE seq >= $1 ORDER BY seq ASC LIMIT $2`
		rows, err := s.db.Query(selectStm, start, count)
		if err != nil {
			return nil, err
		}
		return &iterator{rows: rows}, nil
	}
}
