package shelltest

import (
	"log"
	"testing"
)

func TestShAssumptions(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	shell, err := newShell(nil)
	if err != nil {
		t.Fatal(err)
	}

	//Intentially not having tabs, some shells will error on tab !: command not found
	res, err := shell.Run(`cd /tmp
X=7
export CYGNUS=blackhole
echo $X
pwd
`)

	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !res.Matches(`7`) {
		t.Errorf("No 7")
	}
	if !res.Matches(`/tmp`) {
		t.Errorf("Not in /tmp")
	}

	//Intentially not having tabs, some shells will error on tab !: command not found
	res, err = shell.Run(`echo $X
pwd
env
`)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !res.Matches(`7`) {
		t.Errorf("No 7")
	}
	if !res.Matches(`/tmp`) {
		t.Errorf("Not in /tmp")
	}
}
