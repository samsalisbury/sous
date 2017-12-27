package storage

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/stretchr/testify/suite"
	// it's a SQL db driver. This is how you do that.
	_ "github.com/lib/pq"
)

type PostgresStateManagerSuite struct {
	suite.Suite
	manager *PostgresStateManager
	db      *sql.DB
}

func (suite *PostgresStateManagerSuite) SetupTest() {
	var err error

	port := "6543"
	if np, set := os.LookupEnv("PGPORT"); set {
		port = np
	}
	connstr := fmt.Sprintf("dbname=sous-test-template host=localhost port=%s", port)
	setupDB, err := sql.Open("postgres", connstr)
	if err != nil {
		suite.FailNow(fmt.Sprintf("Error setting up test database: %v. Did you already `make postgres-test-prepare`?", err))
	}
	// ignoring error because I think "no such DB is a failure"
	setupDB.Exec("drop database sous-test")
	if _, err := setupDB.Exec("create database sous-test template sous-test-template"); err != nil {
		suite.FailNow(fmt.Sprintf("Error creating test database: %v", err))
	}
	if err := setupDB.Close(); err != nil {
		suite.FailNow(fmt.Sprintf("Error closing DB manipulation connection: %v", err))
	}

	suite.manager, err = NewPostgresStateManager(PostgresConfig{
		DBName:   "sous-test",
		User:     "",
		Password: "",
		Host:     "localhost",
		Port:     port,
	})

	connstr = fmt.Sprintf("dbname=sous-test host=localhost port=%s", port)
	if suite.db, err = sql.Open("postgres", connstr); err != nil {
		suite.FailNow(fmt.Sprintf("Error establishing test-assertion DB connection: %v", err))
	}
}

func (suite *PostgresStateManagerSuite) TestWriteState_success(t *testing.T) {
	s := exampleState()

	suite.NoError(suite.manager.WriteState(s, testUser))
	suite.NoError(suite.manager.WriteState(s, testUser))

	ns, err := suite.manager.ReadState()
	suite.NoError(err)

	oldD, err := s.Deployments()
	suite.NoError(err)
	newD, err := ns.Deployments()

	for diff := range oldD.Diff(newD).Pairs {
		switch diff.Kind() {
		default:
			suite.Fail("Difference detected between written and read states: %#v", diff)
		case sous.SameKind:
		}
	}
}

func TestPostgresStateManager(t *testing.T) {
	suite.Run(t, new(PostgresStateManagerSuite))
}
