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
