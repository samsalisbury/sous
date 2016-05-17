package tests

import (
	"log"
	"testing"

	"github.com/opentable/sous/cli"
	"github.com/opentable/sous/util/cmdr"
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
)

func TestSous(t *testing.T) {

	term := NewTerminal(t, &cli.Sous{})

	// Invoke the CLI
	term.RunCommand("sous")

	log.Print(term.Stderr)
	term.Stdout.ShouldHaveNumLines(0)
	term.Stderr.ShouldHaveNumLines(19)

	term.Stderr.ShouldHaveExactLine("usage: sous <command>")
	term.Stderr.ShouldHaveLineContaining("help     get help with sous")
}

func TestSousVersion(t *testing.T) {

	term := NewTerminal(t, &cli.Sous{})

	term.CLI.Hooks.PreExecute = func(c cmdr.Command) error {
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
