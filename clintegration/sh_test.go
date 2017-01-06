package clintegration

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/opentable/sous/util/shelltest"
)

func TestShAssumptions(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	shell, err := NewShell(nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := shell.Run(`
	cd /tmp
	X=7
	export CYGNUS=blackhole
	echo $X
	pwd
	`)

	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !res.OutputMatches(`7`) {
		t.Errorf("No 7")
	}
	if !res.OutputMatches(`/tmp`) {
		t.Errorf("Not in /tmp")
	}

	res, err = shell.Run(`
	echo $X
	pwd
	env
	`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !res.OutputMatches(`7`) {
		t.Errorf("No 7")
	}
	if !res.OutputMatches(`/tmp`) {
		t.Errorf("Not in /tmp")
	}
}

func SomethingSomething(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Couldn't find PWD: %s", err)
	}

	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("Couldn't find go: %s (really? how are you running this test then?)", err)
	}

	goDir := filepath.Dir(goExe)

	testHome := "integration/test-homedir"

	shell := shelltest.New(t, map[string]string{
		"HOME":    filepath.Join(pwd, testHome),
		"GIT_SSH": "ssh_wrapper",
		"GOPATH":  filepath.Join(pwd, testHome, "golang"),
		"PATH":    strings.Join([]string{"~/bin", goDir, filepath.Join(pwd, testHome, "golang/bin")}, ':'),
	})

	prologue := shell.Block("Test environment setup", `
	source ~/.bashrc
	go get github.com/nyarly/cygnus
	`)

	setup := prologue.Block("sous setup", `
	git clone our.git.server/sous-server
	cd sous-server
	sous build
	SOUS_SERVER= sous deploy -cluster one-left,one-right,two`,
		func(res Result, t thing) {
			t.CrashIf(res.Exit != 0)
			t.ErrorIf(!res.OutputMatches(`Deployed`), "No report of deployment")
		})

	config := setup.Block("configuration", `
	sous config
	`)

	deploy := config.Block("deploy project", `
	git clone our.git.server/test-project
	cd test-project
	sous init
	sous build
	sous deploy
	`)

	check := deploy.Block("confirm deployment", `
	cygnus -x=1
	`)
}
