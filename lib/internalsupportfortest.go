package sous

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/lib/pq"
)

// SetupDB makes a test DB in a local postgres database. It assumes the
// existence of a properly migrated 'sous_test_template' directory. Each test
// should provide a unique name for its DB instance so that they'll be
// independent.
func SetupDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := setupDBErr(name)
	if err != nil {
		t.Skipf("setupDB failed: %s", err)
	}
	return db
}

func setupDBErr(name string) (*sql.DB, error) {
	port := "6543"
	if np, set := os.LookupEnv("PGPORT"); set {
		port = np
	}
	dbName := "sous_test_" + name
	if dbName == "sous_test_template" {
		return nil, fmt.Errorf("Cannot use test name %q because the DB name %q is used as the template.", name, dbName)
	}
	connstr := fmt.Sprintf("dbname=sous_test_template host=localhost port=%s sslmode=disable", port)
	setupDB, err := sql.Open("postgres", connstr)
	if err != nil {
		return nil, fmt.Errorf("Error setting up test database %q Error: %v. Did you already `make postgres-test-prepare`?", dbName, err)
	}
	if _, err := setupDB.Exec("drop database " + dbName); err != nil && !isNoDBError(err) {
		return nil, fmt.Errorf("Error dropping old test database %q: connstr %q err %v", dbName, connstr, err)
	}
	if _, err := setupDB.Exec("create database " + dbName + " template sous_test_template"); err != nil {
		return nil, fmt.Errorf("Error creating test database connstr %q err %v", connstr, err)
	}
	if err := setupDB.Close(); err != nil {
		return nil, fmt.Errorf("Error closing DB manipulation connection connstr %q err %v", connstr, err)
	}

	connstr = fmt.Sprintf("dbname=%s host=localhost port=%s sslmode=disable", dbName, port)

	db, err := sql.Open("postgres", connstr)
	if err != nil {
		return nil, fmt.Errorf("Creating test sql.DB, error: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func isNoDBError(err error) bool {
	pqerr, is := err.(*pq.Error)
	if !is {
		return false
	}
	return pqerr.Code == "3D000" // invalid_catalog_name per https://www.postgresql.org/docs/current/static/errcodes-appendix.html
}
