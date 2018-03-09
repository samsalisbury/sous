package storage

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/require"
)

func TestDuplexWrite(t *testing.T) {
	s := exampleState()

	clobberDir(t, "testdata/result")
	PrepareTestGitRepo(t, s, "testdata/remote", "testdata/out")

	db := setupDB(t)
	defer db.Close()

	log, _ := logging.NewLogSinkSpy()
	gsm := NewGitStateManager(NewDiskStateManager("testdata/out"))
	psm := NewPostgresStateManager(db, log)
	dupsm := NewDuplexStateManager(gsm, psm, log)

	require.NoError(t, dupsm.WriteState(s, testUser))

	pstate, err := psm.ReadState()
	require.NoError(t, err)
	assertStatesEqual(t, s, pstate)

	remoteAbs, err := filepath.Abs("testdata/remote")
	if err != nil {
		t.Fatal(err)
	}
	runCmd(t, "testdata", "git", "clone", "file://"+remoteAbs, "result")

	os.RemoveAll("testdata/result/.git")

	d := exec.Command("diff", "-r", "testdata/in", "testdata/result")

	if out, err := d.CombinedOutput(); err != nil {
		t.Fatalf("Output not as expected: %s;\n%s", err, string(out))
	}
}

func TestDuplexReadState(t *testing.T) {
	s := exampleState()
	PrepareTestGitRepo(t, s, "testdata/remote", "testdata/out")

	db := setupDB(t)
	defer db.Close()
	log, _ := logging.NewLogSinkSpy()
	gsm := NewGitStateManager(NewDiskStateManager("testdata/out"))
	psm := NewPostgresStateManager(db, log)

	dupsm := NewDuplexStateManager(gsm, psm, log)

	actual, err := dupsm.ReadState()
	require.NoError(t, err)

	expected := exampleState()

	sameYAML(t, actual, expected)

	pstate, err := psm.ReadState()
	require.NoError(t, err)
	assertStatesEqual(t, expected, pstate)
}

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := setupDBErr()
	if err != nil {
		t.Skipf("setupDB failed: %s", err)
	}
	return db
}

func setupDBErr() (*sql.DB, error) {
	port := "6543"
	if np, set := os.LookupEnv("PGPORT"); set {
		port = np
	}
	connstr := fmt.Sprintf("dbname=sous_test_template host=localhost port=%s user=postgres sslmode=disable", port)
	setupDB, err := sql.Open("postgres", connstr)
	if err != nil {
		return nil, fmt.Errorf("Error setting up test database Error: %v. Did you already `make postgres-test-prepare`?", err)
	}
	if _, err := setupDB.Exec("drop database sous_test"); err != nil && !isNoDBError(err) {
		return nil, fmt.Errorf("Error dropping old test database connstr %q err %v", connstr, err)
	}
	if _, err := setupDB.Exec("create database sous_test template sous_test_template"); err != nil {
		return nil, fmt.Errorf("Error creating test database connstr %q err %v", connstr, err)
	}
	if err := setupDB.Close(); err != nil {
		return nil, fmt.Errorf("Error closing DB manipulation connection connstr %q err %v", connstr, err)
	}
	db, err := PostgresConfig{
		DBName:   "sous_test",
		User:     "postgres",
		Password: "",
		Host:     "localhost",
		Port:     port,
		SSL:      false,
	}.DB()

	if err != nil {
		return nil, fmt.Errorf("Creating test sql.DB, error: %v", err)
	}
	return db, nil
}

func assertStatesEqual(t *testing.T, oldState, newState *sous.State) {
	t.Helper()

	oldD, err := oldState.Deployments()
	require.NoError(t, err)
	newD, err := newState.Deployments()
	require.NoError(t, err)

	for diff := range oldD.Diff(newD).Pairs {
		switch diff.Kind() {
		default:
			t.Errorf("Difference detected between written and read states: %s %+#v", diff.Kind(), diff)
		case sous.ModifiedKind:
			t.Errorf("Difference detected between modified written and read states: %+#v %+#v", diff, diff.Diffs())

		case sous.SameKind:
		}
	}
}
