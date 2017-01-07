package clintegration

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
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

func buildPath(exes ...string) (string, error) {
	dirMap := map[string]struct{}{}

	for _, name := range exes {
		exePath, err := exec.LookPath(name)
		if err != nil {
			return "", err
		}

		dirMap[filepath.Dir(exePath)]
	}

	dirs := []string{}
	for path := range dirMap {
		dirs = append(dirs, path)
	}

	return strings.Join(dirs, ":")
}

func SomethingSomething(t *testing.T) {
	descPath := os.Getenv("SOUS_QA_DESC")
	if descPath == "" {
		t.Fatalf("SOUS_QA_DESC is empty - you need to run sous_qa_setup and set that env var")
	}

	envDesc, err := desc.LoadDesc(descPath)
	if err != nil {
		t.Fatalf("Couldn't load a QA env description from SOUS_QA_DESC(%q): %s", descPath, err)
	}

	pwd := filepath.Dir(descPath)

	exePATH, err := buildPath("go", "git", "ssh")

	testHome := "integration/test-homedir"

	shell := shelltest.New(t, map[string]string{
		"HOME":    filepath.Join(pwd, testHome),
		"GIT_SSH": "ssh_wrapper",
		"GOPATH":  filepath.Join(pwd, testHome, "golang"),
		"PATH":    strings.Join([]string{"~/bin", exePATH, filepath.Join(pwd, testHome, "golang/bin")}, ':'),
	})

	prologue := shell.Block("Test environment setup", `
	# source ~/.bashrc
	go get github.com/nyarly/cygnus
	go install `+pwd+` #install the current sous project
	cp integration/test-registry/git-server/git_pubkey_rsa* ~/dot-ssh/
	chmod go-rwx -R ~/dot-ssh
	`)

	// TEMPLATING CONFIG STUFF FROM envDesc GOES HERE

	setup := prologue.Block("sous setup", `
	git clone ssh://root@`+envDesc.GitOrigin+`/sous-server
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
