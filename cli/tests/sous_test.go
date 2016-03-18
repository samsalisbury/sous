package tests

import "testing"

func TestSous(t *testing.T) {

	term := NewTerminal(t)

	// Invoke the CLI
	term.RunCommand("sous")

	term.Stdout.ShouldHaveNumLines(0)
	term.Stderr.ShouldHaveNumLines(2)

	term.Stderr.ShouldHaveExactLine("usage: sous [options] command")
	term.Stderr.ShouldHaveLineContaining(
		"try `sous help` for a list of commands")
}

func TestSousVersion(t *testing.T) {
	term := NewTerminal(t)
	term.RunCommand("sous version")
	term.Stderr.ShouldHaveNumLines(0)
	term.Stdout.ShouldHaveNumLines(1)
	term.Stdout.ShouldHaveExactLine("sous version 0.0.0-test")
}
