package sql_test

import (
	"context"
	gosql "database/sql"
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
	dsn, closer, err := postgresServer()
	if err != nil {
		t.Fatal(err)
	}
	defer closer()

	f := func() (core.EventStore, func(), error) {
		es, err := postgreConnect(dsn)
		if err != nil {
			return nil, nil, err
		}
		return es, es.Close, nil
	}
	testsuite.Test(t, f)
}

func TestFetchFuncAllPostgres(t *testing.T) {
	dsn, closer, err := postgresServer()
	if err != nil {
		t.Fatal(err)
	}
	defer closer()
	es, err := postgreConnect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer es.Close()
	testsuite.TestFetcherAll(t, es, es.All(0))
}

func postgresServer() (string, func(), error) {
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
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}
	// Start the container
	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", nil, err
	}

	// Get container host and port
	host, _ := postgresContainer.Host(ctx)
	port, _ := postgresContainer.MappedPort(ctx, "5432")

	// Build the DSN
	dsn := fmt.Sprintf("host=%s port=%s user=test password=secret dbname=testdb sslmode=disable", host, port.Port())
	return dsn, func() { postgresContainer.Terminate(ctx) }, nil

}

func postgreConnect(dsn string) (*sql.Postgres, error) {
	// Connect using database/sql
	db, err := gosql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db open failed: %w", err)
	}
	// Test the connection
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return sql.NewPostgres(db)
}
