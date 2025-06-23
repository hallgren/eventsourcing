package sql_test

import (
	"context"
	sqldriver "database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/core/testsuite"
	"github.com/hallgren/eventsourcing/eventstore/sql"
)

func TestSuitePostgres(t *testing.T) {
	ctx := context.Background()

	// Set up the PostgreSQL container request
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16", // Use a specific version
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithStartupTimeout(30 * time.Second),
	}

	// Start the container
	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer postgresContainer.Terminate(ctx)

	// Get container host and port
	host, _ := postgresContainer.Host(ctx)
	port, _ := postgresContainer.MappedPort(ctx, "5432")

	// Build the DSN
	dsn := fmt.Sprintf("host=%s port=%s user=test password=secret dbname=testdb sslmode=disable", host, port.Port())

	f := func() (core.EventStore, func(), error) {
		// Connect using database/sql
		db, err := sqldriver.Open("postgres", dsn)
		if err != nil {
			return nil, nil, err
		}
		// Test the connection
		err = db.Ping()
		if err != nil {
			return nil, nil, err
		}
		es := sql.Open(db)
		err = es.MigratePostgres()
		if err != nil {
			return nil, nil, err
		}
		return es, func() {
			db.Close()
		}, nil
	}
	testsuite.Test(t, f)
}
