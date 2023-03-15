package memory_test

import (
	"testing"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/eventstore/memory"
	"github.com/hallgren/eventsourcing/eventstore/suite"
)

func TestSuite(t *testing.T) {
	f := func(ser eventsourcing.Serializer[suite.FrequentFlierEvent]) (eventsourcing.EventStore[suite.FrequentFlierEvent], func(), error) {
		es := memory.Create[suite.FrequentFlierEvent]()
		return es, func() { es.Close() }, nil
	}

	suite.Test[suite.FrequentFlierEvent](t, f)
}
