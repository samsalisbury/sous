package storage

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/nyarly/testify/require"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
)

func TestGitWriteState(t *testing.T) {

	s := exampleState()

	if err := os.RemoveAll("testdata/out"); err != nil {
		t.Fatal(err)
	}

	gsm := NewGitStateManager(NewDiskStateManager("testdata/out"))

	if err := gsm.WriteState(s); err != nil {
		t.Fatal(err)
	}

	d := exec.Command("diff", "-r", "testdata/in", "testdata/out")
	out, err := d.CombinedOutput()
	if err != nil {
		t.Log("Output not as expected:")
		t.Log(string(out))
		t.Fatal("")
	}
}

func TestGitReadState(t *testing.T) {
	require := require.New(t)

	gsm := NewGitStateManager(NewDiskStateManager("testdata/in"))

	actual, err := gsm.ReadState()
	require.NoError(err)

	expected := exampleState()

	sameYAML(t, actual, expected)
}

func sameYAML(t *testing.T, actual *sous.State, expected *sous.State) {
	assert := assert.New(t)
	require := require.New(t)

	ams := actual.Manifests.Snapshot()
	ems := expected.Manifests.Snapshot()
	assert.Len(ams, len(ems))
	for n, v := range ems {
		assert.Contains(ams, n)
		assert.Equal(ams[n], v)
	}

	actualYAML, err := yaml.Marshal(actual)
	require.NoError(err)
	expectedYAML, err := yaml.Marshal(expected)
	require.NoError(err)
	assert.Equal(actualYAML, expectedYAML)
}

func runScript(t *testing.T, script string, dir ...string) {
	lines := strings.Split(script, "\n")
	for _, l := range lines {
		words := strings.Split(strings.Trim(l, " \t"), " ")
		cmd := exec.Command(words[0], words[1:]...)
		if len(dir) > 0 {
			cmd.Dir = dir[0]
		}
		cmd.Env = []string{"GIT_CONFIG_NOSYSTEM=true", "HOME=none", "XDG_CONFIG_HOME=none"}
		log.Print(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}
	}
}

func TestGitUpdates(t *testing.T) {
	require := require.New(t)

	runScript(t, `rm -rf testdata/origin testdata/target
	cp -a testdata/in testdata/origin`)
	runScript(t, `git init
	git add .
	git commit -m ""`, `testdata/origin`)
	runScript(t, `git clone origin target`, `testdata`)

	gsm := NewGitStateManager(NewDiskStateManager("testdata/target"))
	dsm := NewDiskStateManager(`testdata/origin`)

	actual, err := gsm.ReadState()
	require.NoError(err)

	expected := exampleState()
	sameYAML(t, actual, expected)

	expected.Manifests.Add(&sous.Manifest{Source: sous.SourceLocation{Repo: "github.com/opentable/brandnew"}})
	dsm.WriteState(expected)
	runScript(t, `git add .
	git commit -m ""`, `testdata/origin`)

	actual, err = gsm.ReadState()
	require.NoError(err)
	sameYAML(t, actual, expected)
}

func TestGitReadState_empty(t *testing.T) {
	gsm := NewGitStateManager(NewDiskStateManager("testdata/nonexistent"))
	actual, err := gsm.ReadState()
	if err != nil && !os.IsNotExist(errors.Cause(err)) {
		t.Fatal(err)
	}
	d, err := actual.Deployments()
	if err != nil {
		t.Fatal(err)
	}
	if d.Len() != 0 {
		t.Errorf("got len %d; want %d", d.Len(), 0)
	}
}
