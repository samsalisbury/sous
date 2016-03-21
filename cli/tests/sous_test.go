package tests

import (
	"testing"

	"github.com/opentable/sous/cli"
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
)

func TestSous(t *testing.T) {

	term := NewTerminal(t, &cli.Sous{})

	// Invoke the CLI
	term.RunCommand("sous")

	term.Stdout.ShouldHaveNumLines(0)
	term.Stderr.ShouldHaveNumLines(2)

	term.Stderr.ShouldHaveExactLine("usage: sous [options] command")
	term.Stderr.ShouldHaveLineContaining(
		"try `sous help` for a list of commands")
}

func TestSousVersion(t *testing.T) {

	term := NewTerminal(t, &cli.Sous{})

	term.CLI.Hooks.PreExecute = func(c cli.Command) error {
		g := psyringe.New()
		g.Fill(
			&cli.Sous{Version: semv.MustParse("1.0.0-test")},
		)
		return g.Inject(c)
	}

	// This prints the whole shell session if the test fails.
	defer term.PrintFailureSummary()

	term.RunCommand("sous version")
	term.Stderr.ShouldHaveNumLines(0)
	term.Stdout.ShouldHaveNumLines(1)
	term.Stdout.ShouldHaveExactLine("sous version 1.0.0-test")
}
