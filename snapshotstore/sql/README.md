# SQL Snapshot Store

The sql is a module containing multiple sql based snapshot stores that are all based on the
database/sql interface in go standard library. The different snapshot stores has specific database schemas
that support the different databases.

## SQLite

Supports the SQLite database https://www.sqlite.org/

### Database Schema

```go
CREATE TABLE IF NOT EXISTS snapshots (
    id              VARCHAR NOT NULL,
    type            VARCHAR,
    version         INTEGER,
    global_version  INTEGER,
    state           BLOB
);

CREATE UNIQUE INDEX IF NOT EXISTS id_type ON snapshots (id, type);
```

### Constructor

```go
// NewSQLite connection to database
func NewSQLite(db *sql.DB) (*SQLite, error) {
```

### Example of use

```go

import (
	sqldriver "database/sql"
	"github.com/hallgren/eventsourcing/snapshotstore/sql"
	_ "github.com/mattn/go-sqlite3"
)

db, err := sqldriver.Open("sqlite3", "file::memory:?locked.sqlite?cache=shared")
if err != nil {
	return nil, nil, err
}

sqliteSnapshotStore, err := sql.NewSQLite(db)
if err != nil {
	return nil, nil, err
}
```
## Postgres

Supports the Postgres database https://www.postgresql.org

### Database Schema

```go
CREATE TABLE IF NOT EXISTS snapshots (
    id VARCHAR NOT NULL,
    type VARCHAR,
    version INTEGER,
    global_version INTEGER,
    state BYTEA
);

CREATE UNIQUE INDEX IF NOT EXISTS id_type ON snapshots (id, type);
```

### Constructor

```go
// NewPostgres connection to database
func NewPostgres(db *sql.DB) (*Postgres, error) {
```

### Example of use

```go
import (
	gosql "database/sql"
	_ "github.com/lib/pq"
	"github.com/hallgren/eventsourcing/snapshotstore/sql"
)

db, err := gosql.Open("postgres", dsn)
if err != nil {
  return err
}


postgreSnapshotStore, err := sql.NewPostgres(db)
if err != nil {
	return err
}
```
