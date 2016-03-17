package cli

import "testing"

func TestBuildGraph(t *testing.T) {

	g, err := BuildGraph()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := g.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}
