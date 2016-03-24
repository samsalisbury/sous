package cli

import (
	"testing"

	"github.com/opentable/sous/util/cmdr"
)

func TestBuildGraph(t *testing.T) {

	g, err := BuildGraph(&Sous{}, &cmdr.CLI{})

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := g.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}
