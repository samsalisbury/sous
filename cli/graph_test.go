package cli

import "testing"

func TestBuildGraph(t *testing.T) {

	c := &Sous{}

	if err := c.buildGraph(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := c.Graph.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}
