package sql_test

import (
	"context"
	gosql "database/sql"
	"fmt"
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
	dsn, closer, err := sqlServer()
	if err != nil {
		t.Fatal(err)
	}
	defer closer()

	f := func() (core.EventStore, func(), error) {
		es, err := sqlServerConnect(dsn)
		if err != nil {
			return nil, nil, err
		}
		return es, es.Close, nil
	}
	testsuite.Test(t, f)
}

func TestFetchFuncAllSQLServer(t *testing.T) {
	dsn, closer, err := sqlServer()
	if err != nil {
		t.Fatal(err)
	}
	defer closer()

	es, err := sqlServerConnect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	testsuite.TestFetcher(t, es, es.All(0))
}

func sqlServerConnect(dsn string) (*sql.SQLServer, error) {
	var db *gosql.DB
	var err error
	for i := 0; i < 10; i++ {
		// Connect using database/sql
		db, err = gosql.Open("sqlserver", dsn)
		// Test the connection
		if err == nil && db.Ping() == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, err
	}
	return sql.NewSQLServer(db)
}

func sqlServer() (string, func(), error) {
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
		return "", nil, err
	}

	host, err := mssqlC.Host(ctx)
	if err != nil {
		return "", nil, err
	}

	port, err := mssqlC.MappedPort(ctx, "1433")
	if err != nil {
		return "", nil, err
	}

	dsn := fmt.Sprintf("sqlserver://sa:YourStrong(!)Password@%s:%s?database=master", host, port.Port())
	return dsn, func() { mssqlC.Terminate(ctx) }, nil
}
