package sous

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/lib/pq"
)

// SetupDB makes a test DB in a local postgres database. It assumes the
// existence of a properly migrated 'sous_test_template' directory. Each test
// should provide a unique name for its DB instance so that they'll be
// independent.
func SetupDB(t *testing.T, optidx ...int) *sql.DB {
	t.Helper()
	name := dbNameRoot(t, optidx...)
	t.Logf("Creating DB for %s called sous_test_%s", t.Name(), name)
	db, err := setupDBErr(name)
	if err != nil {
		if os.Getenv("SOUS_TEST_NODB") != "" {
			t.Skipf("setupDB failed: %s", err)
			return nil
		}
		t.Fatalf("Error creating test DB: %v (Set SOUS_TEST_NODB=yes) to skip tests that rely on the DB", err)
	}
	return db
}

// DBNameForTest returns a database name based on the test name.
func DBNameForTest(t *testing.T, optidx ...int) string {
	t.Helper()
	return "sous_test_" + dbNameRoot(t, optidx...)
}

// ReleaseDB should be called in any test that called SetupDB (even indirectly)
func ReleaseDB(t *testing.T, optidx ...int) {
	name := dbNameRoot(t, optidx...)
	if db, has := dbconns[name]; has {
		db.Close() //ignoring error
		delete(dbconns, name)
	}
}

func dbNameRoot(t *testing.T, optidx ...int) string {
	name := strings.Replace(strings.ToLower(t.Name()), "test", "", -1)
	name = strings.Replace(name, "/", "_", -1)
	name = strings.Replace(name, "-", "_", -1)
	name = strings.Replace(name, ":", "_", -1)
	if len(optidx) > 0 {
		return fmt.Sprintf("%s_%d", name, optidx[0])
	}
	return name
}

var dbconns = map[string]*sql.DB{}
var adminConn *sql.DB

var setupAdminConn = sync.Once{}

func getAdminConn() (*sql.DB, error) {
	var err error
	setupAdminConn.Do(func() {
		port := "6543"
		if np, set := os.LookupEnv("PGPORT"); set {
			port = np
		}
		connstr := fmt.Sprintf("dbname=sous_test_template host=localhost user=postgres port=%s sslmode=disable", port)
		adminConn, err = sql.Open("postgres", connstr)
	})
	if err != nil {
		return nil, err
	}
	if adminConn == nil {
		return nil, fmt.Errorf("no admin SQL connection")
	}
	return adminConn, err
}

var dbsetupMutex = sync.Mutex{}

func setupDBErr(name string) (*sql.DB, error) {
	dbsetupMutex.Lock()
	defer dbsetupMutex.Unlock()
	port := "6543"
	if np, set := os.LookupEnv("PGPORT"); set {
		port = np
	}
	dbName := "sous_test_" + name
	if dbName == "sous_test_template" {
		return nil, fmt.Errorf("cannot use test name %q because the DB name %q is used as the template", name, dbName)
	}
	setupDB, err := getAdminConn()

	if err != nil {
		return nil, fmt.Errorf("error setting up test database %q Error: %v. Did you already `make postgres-test-prepare`?", dbName, err)
	}
	if _, err := setupDB.Exec("drop database " + dbName); err != nil && !isNoDBError(err) {
		return nil, fmt.Errorf("error dropping old test database %q: err %v", dbName, err)
	}
	if _, err := setupDB.Exec("create database " + dbName + " template sous_test_template"); err != nil {
		return nil, fmt.Errorf("error creating test database err %v", err)
	}

	connstr := fmt.Sprintf("dbname=%s host=localhost user=postgres port=%s sslmode=disable", dbName, port)

	db, err := sql.Open("postgres", connstr)
	if err != nil {
		return nil, fmt.Errorf("connecting to test sql.DB, error: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("checking connection to DB at %q: %v", connstr, err)
	}
	dbconns[name] = db

	return db, nil
}

func isNoDBError(err error) bool {
	pqerr, is := err.(*pq.Error)
	if !is {
		return false
	}
	return pqerr.Code == "3D000" // invalid_catalog_name per https://www.postgresql.org/docs/current/static/errcodes-appendix.html
}
