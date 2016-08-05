package cli

import (
	"testing"

	"github.com/opentable/sous/util/cmdr"
)

func TestBuildGraph(t *testing.T) {

	g := BuildGraph(&Sous{}, &cmdr.CLI{})

	if err := g.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}
