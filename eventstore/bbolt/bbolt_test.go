package bbolt_test

import (
	"os"
	"testing"

	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/core/testsuite"
	"github.com/hallgren/eventsourcing/eventstore/bbolt"
)

func TestSuite(t *testing.T) {
	f := func() (core.EventStore, func(), error) {
		dbFile := "bolt.db"
		es, err := bbolt.New(dbFile)
		if err != nil {
			return nil, nil, err
		}
		return es, func() {
			es.Close()
			os.Remove(dbFile)
		}, nil
	}
	testsuite.Test(t, f)
}
