package kurrent_test

import (
	"context"
	"testing"

	"github.com/kurrent-io/KurrentDB-Client-Go/kurrentdb"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/core/testsuite"
	es "github.com/hallgren/eventsourcing/eventstore/kurrent"
)

func TestSuite(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "kurrentplatform/kurrentdb:latest",
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
		settings, err := kurrentdb.ParseConnectionString(endpoint + "?tls=false")
		if err != nil {
			return nil, nil, err
		}

		db, err := kurrentdb.NewClient(settings)
		if err != nil {
			return nil, nil, err
		}

		es := es.Open(db, true)
		return es, func() {
		}, nil
	}
	testsuite.Test(t, f)
}
