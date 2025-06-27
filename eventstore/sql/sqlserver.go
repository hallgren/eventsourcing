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

var stmSQLServer = []string{
	`create table events (seq INTEGER PRIMARY KEY AUTOINCREMENT, id VARCHAR NOT NULL, version INTEGER, reason VARCHAR, type VARCHAR, timestamp VARCHAR, data BLOB, metadata BLOB);`,
	`create unique index id_type_version on events (id, type, version);`,
	`create index id_type on events (id, type);`,
}

// SQLite event store handler
type SQLServer struct {
	db   *sql.DB
	lock *sync.Mutex
}

// NewSQLite connection to database
func NewSQLServer(db *sql.DB) (*SQLServer, error) {
	if err := migrate(db, stmSQLServer); err != nil {
		return nil, err
	}
	return &SQLServer{
		db: db,
	}, nil
}

// Close the connection
func (s *SQLServer) Close() {
	s.db.Close()
}

// Save persists events to the database
func (s *SQLServer) Save(events []core.Event) error {
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
	selectStm := `Select version from events where id=? and type=? order by version desc limit 1`
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
func (s *SQLServer) Get(ctx context.Context, id string, aggregateType string, afterVersion core.Version) (core.Iterator, error) {
	selectStm := `Select seq, id, version, reason, type, timestamp, data, metadata from events where id=? and type=? and version>? order by version asc`
	rows, err := s.db.QueryContext(ctx, selectStm, id, aggregateType, afterVersion)
	if err != nil {
		return nil, err
	}
	return &iterator{rows: rows}, nil
}

// All iterate over all event in GlobalEvents order
func (s *SQLServer) All(start core.Version, count uint64) (core.Iterator, error) {
	selectStm := `Select seq, id, version, reason, type, timestamp, data, metadata from events where seq >= ? order by seq asc LIMIT ?`
	rows, err := s.db.Query(selectStm, start, count)
	if err != nil {
		return nil, err
	}
	return &iterator{rows: rows}, nil
}
