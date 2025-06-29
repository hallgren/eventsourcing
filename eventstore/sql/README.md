# SQL Event Store

The sql eventstore is a module containing multiple sql based event stores that are all based on the
database/sql interface in go standard library. The different event stores has specific database schemas
that support the dirrent databases.

## SQLite

Supports the SQLite database https://www.sqlite.org/

### Database Schema

```go
    CREATE TABLE IF NOT EXISTS events (
        seq        INTEGER PRIMARY KEY AUTOINCREMENT,
        id         VARCHAR NOT NULL,
        version    INTEGER,
        reason     VARCHAR,
        type       VARCHAR,
        timestamp  VARCHAR,
        data       BLOB,
        metadata   BLOB,
        UNIQUE (id, type, version)
    );

    CREATE INDEX IF NOT EXISTS id_type ON events (id, type);
```

### Constructor

// NewSQLite connection to database
NewSQLite(db *sql.DB) (*SQLite, error) 

// NewSQLiteSingelWriter prevents multiple writers to save events concurrently
//
// Multiple go routines writing concurrently to sqlite could produce sqlite to lock.
// https://www.sqlite.org/cgi/src/doc/begin-concurrent/doc/begin_concurrent.md
//
// "If there is significant contention for the writer lock, this mechanism can
// be inefficient. In this case it is better for the application to use a mutex
// or some other mechanism that supports blocking to ensure that at most one
// writer is attempting to COMMIT a BEGIN CONCURRENT transaction at a time.
// This is usually easier if all writers are part of the same operating system process."
NewSQLiteSingelWriter(db *sql.DB) (*SQLite, error)

### Example of use

```go

import (
  // have to alias the sql package as it use the same name
	gosql "database/sql"
	"github.com/hallgren/eventsourcing/eventstore/sql"
  // use the sqlite driver from mattn in this example
	_ "github.com/mattn/go-sqlite3"
)


db, err := gosql.Open("sqlite3", "file::memory:?cache=shared")
if err != nil {
	return nil, nil, errors.New(fmt.Sprintf("could not open database %v", err))
}
err = db.Ping()
if err != nil {
	return nil, nil, errors.New(fmt.Sprintf("could not ping database %v", err))
}
sqliteEventStore, err := sql.NewSQLiteSingelWriter(db)
if err != nil {
	return nil, nil, err
}
```

## Postgres

Supports the Postgres database https://www.postgresql.org

### Database Schema

```go
CREATE TABLE IF NOT EXISTS events (
	seq SERIAL PRIMARY KEY,
	id VARCHAR NOT NULL,
	version INTEGER,
	reason VARCHAR,
	type VARCHAR,
	timestamp VARCHAR,
	data BYTEA,
	metadata BYTEA,
	UNIQUE (id, type, version)
);

CREATE INDEX IF NOT EXISTS id_type ON events (id, type);
```

### Constructor

// NewPostgres connection to database
func NewPostgres(db *sql.DB) (*Postgres, error) {

### Example of use

```go

import (

  // have to alias the sql package as it use the same name
	gosql "database/sql"

  // in this example we use the pg postgres driver
	_ "github.com/lib/pq"
	"github.com/hallgren/eventsourcing/eventstore/sql"
)

db, err := gosql.Open("postgres", dsn)
if err != nil {
	return nil, nil, fmt.Errorf("db open failed: %w", err)
}
// Test the connection
err = db.Ping()
if err != nil {
	return nil, nil, err
}
postgresEventStore, err := sql.NewPostgres(db)
```
