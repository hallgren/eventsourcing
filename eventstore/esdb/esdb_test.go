//go:build manual
// +build manual

// make these tests manual as they are dependent on a running event store db.

package esdb_test

import (
	"testing"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
	"github.com/hallgren/eventsourcing"
	es "github.com/hallgren/eventsourcing/eventstore/esdb"
	"github.com/hallgren/eventsourcing/eventstore/suite"
)

func TestSuite(t *testing.T) {
	f := func(ser eventsourcing.Serializer[suite.FrequentFlierEvent]) (eventsourcing.EventStore[suite.FrequentFlierEvent], func(), error) {
		// region createClient
		settings, err := esdb.ParseConnectionString("esdb://localhost:2113?tls=false")
		if err != nil {
			return nil, nil, err
		}

		db, err := esdb.NewClient(settings)
		if err != nil {
			return nil, nil, err
		}

		es := es.Open(db, ser, true)
		return es, func() {
		}, nil
	}
	suite.Test[suite.FrequentFlierEvent](t, f)
}
