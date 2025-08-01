# SQL Event Store

The sql eventstore is a module containing multiple sql based event stores that are all based on the
database/sql interface in go standard library. The different event stores has specific database schemas
that support the different databases.

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

```go
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
```

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

```go
// NewPostgres connection to database
func NewPostgres(db *sql.DB) (*Postgres, error) {
```

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

## Microsoft SQL Server

Supports Microsoft SQL Server database https://www.microsoft.com/en-us/sql-server

### Database Schema

```go
IF OBJECT_ID('[events]', 'U') IS NULL
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
END

IF NOT EXISTS (
    SELECT 1 
    FROM sys.indexes 
    WHERE name = 'id_type' AND object_id = OBJECT_ID('events')
)
BEGIN
    CREATE INDEX id_type ON [events] ([id], [type]);
END

IF NOT EXISTS (
    SELECT 1 
    FROM sys.indexes 
    WHERE name = 'id_type' AND object_id = OBJECT_ID('events')
)
BEGIN
    CREATE INDEX id_type ON [events] ([id], [type]);
END
```

### Constructor

```go
// NewSQLServer connection to database
func NewSQLServer(db *sql.DB) (*SQLServer, error) {
```

### Example of use

```go
import (
	// alias the go sql package
	gosql "database/sql"
	 // uses the sql server driver from denisenkom
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/hallgren/eventsourcing/eventstore/sql"
)

db, err := gosql.Open("sqlserver", dsn)
if err != nil {
	return err
}
err = db.Ping() {
if err != nil {
	return err
}
SqlServerEventStore, err := sql.NewSQLServer(db)
```
