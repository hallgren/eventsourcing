package sql_test

import (
	"context"
	gosql "database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/core/testsuite"
	"github.com/hallgren/eventsourcing/eventstore/sql"
)

func TestSuiteSQLServer(t *testing.T) {
	ctx := context.Background()

	// Start MSSQL container
	req := testcontainers.ContainerRequest{
		Image:        "mcr.microsoft.com/mssql/server:2019-latest",
		ExposedPorts: []string{"1433/tcp"},
		Env: map[string]string{
			"ACCEPT_EULA": "Y",
			"SA_PASSWORD": "YourStrong(!)Password",
		},
		WaitingFor: wait.ForLog("SQL Server is now ready for client connections").WithStartupTimeout(2 * time.Minute),
	}

	mssqlC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer func() {
		if err := mssqlC.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate container: %v", err)
		}
	}()

	host, err := mssqlC.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get host: %v", err)
	}

	port, err := mssqlC.MappedPort(ctx, "1433")
	if err != nil {
		t.Fatalf("Failed to get mapped port: %v", err)
	}

	dsn := fmt.Sprintf("sqlserver://sa:YourStrong(!)Password@%s:%s?database=master", host, port.Port())

	f := func() (core.EventStore, func(), error) {
		var db *gosql.DB
		var err2 error
		for i := 0; i < 10; i++ {
			// Connect using database/sql
			db, err2 = gosql.Open("sqlserver", dsn)
			// Test the connection
			if err2 == nil && db.Ping() == nil {
				break
			}
			time.Sleep(2 * time.Second)
		}
		if err2 != nil {
			t.Fatal("Failed to connect to database:", err)
		}
		es, err := sql.NewPostgres(db)
		if err != nil {
			t.Fatal(err)
		}
		return es, func() {
			db.Close()
		}, nil
	}
	testsuite.Test(t, f)
}
