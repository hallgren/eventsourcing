package sql

import (
	"context"
	"fmt"
)

const createTableSQLite = `create table events (seq INTEGER PRIMARY KEY AUTOINCREMENT, id VARCHAR NOT NULL, version INTEGER, reason VARCHAR, type VARCHAR, timestamp VARCHAR, data BLOB, metadata BLOB);`
const createTablePostgres = `CREATE TABLE events (
  seq INTEGER PRIMARY KEY,
  id TEXT NOT NULL,
  version INTEGER,
  reason TEXT,
  type TEXT,
  timestamp TEXT,
  data BYTEA,
  metadata BYTEA
);`

// Migrate is the legacy function that creates the database for sqlite
func (s *SQL) Migrate() error {
	return s.MigrateSQLite()
}

// MigrateSQLite creates the the database for sqlite
func (s *SQL) MigrateSQLite() error {
	sqlStmt := []string{
		createTableSQLite,
		`create unique index id_type_version on events (id, type, version);`,
		`create index id_type on events (id, type);`,
	}
	return s.migrate(sqlStmt)
}

// Migrate the database for Postgres
func (s *SQL) MigratePostgres() error {
	sqlStmt := []string{
		createTablePostgres,
		`create unique index id_type_version on events (id, type, version);`,
		`create index id_type on events (id, type);`,
	}
	return s.migrate(sqlStmt)
}

func (s *SQL) migrate(stm []string) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// check if the migration is already done
	rows, err := tx.Query(`Select count(*) from events`)
	if err == nil {
		rows.Close()
		return nil
	}

	for _, b := range stm {
		_, err := tx.Exec(b)
		if err != nil {
			return fmt.Errorf("tx.Exec failed: %w", err)
		}
	}
	return tx.Commit()
}
