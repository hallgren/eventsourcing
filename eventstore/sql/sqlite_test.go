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

func TestSuite(t *testing.T) {
	f := func() (core.EventStore, func(), error) {
		return eventstore(false)
	}
	testsuite.Test(t, f)
}

func TestSuiteSingelWriter(t *testing.T) {
	f := func() (core.EventStore, func(), error) {
		return eventstore(true)
	}
	testsuite.Test(t, f)
}

func TestMultipleMigrate(t *testing.T) {
	es, close, err := eventstore(false)
	if err != nil {
		t.Fatal(err)
	}
	defer close()
	err = es.Migrate()
	if err != nil {
		t.Fatal(err)
	}
}

func eventstore(singelWriter bool) (*sql.SQL, func(), error) {
	var es *sql.SQL
	db, err := sqldriver.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("could not open database %v", err))
	}
	err = db.Ping()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("could not ping database %v", err))
	}

	if singelWriter {
		es = sql.OpenWithSingelWriter(db)
	} else {
		// to make the concurrent test pass (not have to use this in the sql.OpenWithSingelWriter constructor)
		db.SetMaxOpenConns(1)
		es = sql.Open(db)
	}
	err = es.Migrate()
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("could not migrate database %v", err))
	}
	return es, func() {
		es.Close()
	}, nil
}
