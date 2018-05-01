package storage

import (
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

	db := sous.SetupDB(t)
	defer sous.ReleaseDB(t)

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

	db := sous.SetupDB(t)
	defer sous.ReleaseDB(t)
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
