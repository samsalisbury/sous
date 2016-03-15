package main

import "testing"

func TestGraph(t *testing.T) {

	deps, err := buildGraph()

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if err := deps.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}
