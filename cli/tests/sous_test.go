package tests

import (
	"log"
	"testing"
)

func TestSous(t *testing.T) {
	term := NewTerminal(t, `0.0.0`)

	// Invoke the CLI
	term.RunCommand("sous")

	log.Print(term.Stderr)
	term.Stdout.ShouldHaveNumLines(0)
	term.Stderr.ShouldHaveNumLines(43)

	term.Stderr.ShouldHaveExactLine("usage: sous <command>")
	term.Stderr.ShouldHaveLineContaining("help      get help with sous")
}

func TestSousVersion(t *testing.T) {
	term := NewTerminal(t, "1.0.0-test")

	// This prints the whole shell session if the test fails.
	defer term.PrintFailureSummary()

	term.RunCommand("sous version")
	term.Stderr.ShouldHaveNumLines(0)
	term.Stdout.ShouldHaveNumLines(1)
	term.Stdout.ShouldHaveLineContaining("sous version")
}
