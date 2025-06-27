package sql

import (
	"context"
	"database/sql"
)

func migrate(db *sql.DB, stm []string) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
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
