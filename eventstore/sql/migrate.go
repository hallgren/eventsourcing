package sql

import "context"

const createTable = `create table events (seq INTEGER PRIMARY KEY AUTOINCREMENT, id VARCHAR NOT NULL, version INTEGER, reason VARCHAR, type VARCHAR, timestamp VARCHAR, data BLOB, metadata BLOB);`

// Migrate the database
func (s *SQL[T]) Migrate() error {
	sqlStmt := []string{
		createTable,
		`create unique index id_type_version on events (id, type, version);`,
		`create index id_type on events (id, type);`,
	}
	return s.migrate(sqlStmt)
}

// MigrateTest remove the index that the test sql driver does not support
func (s *SQL[T]) MigrateTest() error {
	return s.migrate([]string{createTable})
}

func (s *SQL[T]) migrate(stm []string) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	for _, b := range stm {
		_, err := tx.Exec(b)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
