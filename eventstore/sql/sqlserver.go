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

const createTableSQLServer = `IF OBJECT_ID('[events]', 'U') IS NULL
BEGIN
    CREATE TABLE [events] (
        [seq] INT IDENTITY(1,1) PRIMARY KEY,
        [id] NVARCHAR(255) NOT NULL,
        [version] INT,
        [reason] NVARCHAR(255),
        [type] NVARCHAR(255),
        [timestamp] NVARCHAR(255),
        [data] VARBINARY(MAX),
        [metadata] VARBINARY(MAX),
        CONSTRAINT uq_events UNIQUE ([id], [type], [version])
    );
END`

const indexSQLServer = `IF NOT EXISTS (
    SELECT 1 
    FROM sys.indexes 
    WHERE name = 'id_type' AND object_id = OBJECT_ID('events')
)
BEGIN
    CREATE INDEX id_type ON [events] ([id], [type]);
END`

var stmSQLServer = []string{
	createTableSQLServer,
	indexSQLServer,
}

// SQLServer event store handler
type SQLServer struct {
	db   *sql.DB
	lock *sync.Mutex
}

// NewSQLServer connection to database
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
	selectStm := `SELECT TOP 1 version FROM [events] WHERE [id] = @id AND [type] = @type ORDER BY version DESC;`
	err = tx.QueryRow(selectStm, sql.Named("id", aggregateID), sql.Named("type", aggregateType)).Scan(&version)
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
	insert := `INSERT INTO [events] (id, version, reason, type, timestamp, data, metadata)
OUTPUT INSERTED.seq
VALUES (@id, @version, @reason, @type, @timestamp, @data, @metadata);`
	for i, event := range events {
		err := tx.QueryRow(
			insert,
			sql.Named("id", event.AggregateID),
			sql.Named("version", event.Version),
			sql.Named("reason", event.Reason),
			sql.Named("type", event.AggregateType),
			sql.Named("timestamp", event.Timestamp.Format(time.RFC3339)),
			sql.Named("data", event.Data),
			sql.Named("metadata", event.Metadata),
		).Scan(&lastInsertedID)
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
	selectStm := `SELECT seq, id, version, reason, type, timestamp, data, metadata
FROM [events]
WHERE id = @id AND type = @type AND version > @version
ORDER BY version ASC;`
	rows, err := s.db.QueryContext(ctx, selectStm, sql.Named("id", id), sql.Named("type", aggregateType), sql.Named("version", afterVersion))
	if err != nil {
		return nil, err
	}
	return &Iterator{Rows: rows}, nil
}

// All iterate over all event in GlobalEvents order
func (s *SQLServer) All(start core.Version) core.FetchFunc {
	return func() (core.Iterator, error) {
		selectStm := `SELECT seq, id, version, reason, type, timestamp, data, metadata
FROM [events]
WHERE seq >= @start
ORDER BY seq ASC;`
		rows, err := s.db.Query(selectStm, sql.Named("start", start))
		if err != nil {
			return nil, err
		}
		return &Iterator{Rows: rows}, nil
	}
}
