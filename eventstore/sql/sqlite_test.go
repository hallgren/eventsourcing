package sql_test

import (
	sqldriver "database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/core/testsuite"
	"github.com/hallgren/eventsourcing/eventstore/sql"
	_ "github.com/mattn/go-sqlite3"
)

func TestSuiteSQLite(t *testing.T) {
	f := func() (core.EventStore, func(), error) {
		return eventstore(false)
	}
	testsuite.Test(t, f)
}

func TestSuiteSQLiteSingelWriter(t *testing.T) {
	f := func() (core.EventStore, func(), error) {
		return eventstore(true)
	}
	testsuite.Test(t, f)
}
func TestFetchFuncAll(t *testing.T) {
	es, close, err := eventstore(false)
	if err != nil {
		t.Fatal(err)
	}
	defer close()
	testsuite.TestFetcherAll(t, es, es.All(0, 10))
}

func eventstore(singelWriter bool) (*sql.SQLite, func(), error) {
	var es *sql.SQLite
	db, err := sqldriver.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("could not open database %v", err))
	}
	err = db.Ping()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("could not ping database %v", err))
	}

	if singelWriter {
		es, err = sql.NewSQLiteSingelWriter(db)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// to make the concurrent test pass (not have to use this in the sql.OpenWithSingelWriter constructor)
		db.SetMaxOpenConns(1)
		es, err = sql.NewSQLite(db)
		if err != nil {
			return nil, nil, err
		}
	}
	return es, func() {
		es.Close()
	}, nil
}
