package esdb_test

import (
	"context"
	"testing"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/core/testsuite"
	es "github.com/hallgren/eventsourcing/eventstore/esdb"
)

func TestSuite(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "eventstore/eventstore:latest",
		ExposedPorts: []string{"2113/tcp"},
		WaitingFor:   wait.ForListeningPort("2113/tcp"),
		Cmd:          []string{"--insecure", "--run-projections=All", "--mem-db"},
	}

	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	defer container.Terminate(ctx)

	endpoint, err := container.PortEndpoint(ctx, "2113", "esdb")
	if err != nil {
		t.Fatal(err)
	}

	f := func() (core.EventStore, func(), error) {
		settings, err := esdb.ParseConnectionString(endpoint + "?tls=false")
		if err != nil {
			return nil, nil, err
		}

		db, err := esdb.NewClient(settings)
		if err != nil {
			return nil, nil, err
		}

		es := es.Open(db, true)
		return es, func() {
		}, nil
	}
	testsuite.Test(t, f)
}
