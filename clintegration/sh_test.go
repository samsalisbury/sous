package clintegration

import (
	"log"
	"testing"
)

func TestShAssumptions(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	shell, err := NewShell()
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
	shell := ShellTest(t)

	shell.First("setup", `
	git clone our.git.server/sous-server
	cd sous-server
	sous build
	SOUS_SERVER= sous deploy -cluster one-left,one-right,two`,
		func(res Result, t thing) {
			t.CrashIf(res.Exit != 0)
			t.ErrorIf(!res.OutputMatches(`Deployed`), "No report of deployment")
		})

	shell.After("setup", "configuration", `
	sous config
	`)

	shell.After("configuration", "deploy project", `
	git clone our.git.server/test-project
	cd test-project
	sous init
	sous build
	sous deploy
	`)

	// When do these get run?
	// I like: as soon as possible,
	// (i.e. After:
	//     when the "before" step has been run => immediately
	//     otherwise => as soon as the before step is run
	// )
	// BUT how do we catch a mispelled "before"?
	//  One option: use return values for "before"s, not strings.

}
