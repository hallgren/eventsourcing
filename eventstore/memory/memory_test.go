package memory_test

import (
	"testing"

	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/core/testsuite"
	"github.com/hallgren/eventsourcing/eventstore/memory"
)

func TestEventStore(t *testing.T) {
	f := func() (core.EventStore, func(), error) {
		es := memory.Create()
		return es, func() { es.Close() }, nil
	}
	testsuite.Test(t, f)
}

func TestFetchFuncAll(t *testing.T) {
	es := memory.Create()
	defer es.Close()
	testsuite.TestFetchFuncAll(t, es, es.All(0, 10))
}
