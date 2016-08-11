package cli

import (
	"io/ioutil"
	"testing"

	"github.com/opentable/sous/util/cmdr"
)

func TestBuildGraph(t *testing.T) {

	g := BuildGraph(&cmdr.CLI{}, ioutil.Discard, ioutil.Discard)
	g.Add(&Sous{})
	g.Add(&DeployFilterFlags{})

	if err := g.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}
