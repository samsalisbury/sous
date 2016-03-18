package cli

import (
	"flag"
	"io"

	"github.com/samsalisbury/semv"
)

// Sous is the main sous command.
type Sous struct {
	// Version is the version of Sous itself.
	Version semv.Version
	// Output is where the return value of commands that have one gets printed.
	Output io.Writer
	// Graph is the dependency injector used to flesh out command dependencies.
	Graph SousCLIGraph
	// flags holds the values of flags passed to this command
	flags struct {
		Help      bool
		Verbosity struct {
			Silent, Quiet, Loud, Debug bool
		}
	}
}

const sousHelp = `
Sous is a tool to help with building, testing, and deploying code.
`

func (c *Sous) Help() string { return sousHelp }

func (c *Sous) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.flags.Help, "-help", false,
		"show help")
	fs.BoolVar(&c.flags.Verbosity.Silent, "s", false,
		"silent verbosity: silence all nonessential output")
	fs.BoolVar(&c.flags.Verbosity.Quiet, "q", false,
		"quiet verbosity: output only essential error messages")
	fs.BoolVar(&c.flags.Verbosity.Loud, "v", false,
		"loud verbosity: output extra info, including all shell commands")
	fs.BoolVar(&c.flags.Verbosity.Debug, "debug", false,
		"debug level verbosity: output detailed logs of internal operations")
}

func (c *Sous) Execute(args []string) Result {
	return UsageError{
		Message: "usage: sous [options] command",
		Tip:     "try `sous help` for a list of commands",
	}
}

func (c *Sous) Subcommands() Commands {
	return Commands{
		"version": &SousVersionCommand{},
		"help":    &SousHelp{},
	}
}

func (c *Sous) Verbosity() Verbosity {
	if c.flags.Verbosity.Debug {
		return Debug
	}
	if c.flags.Verbosity.Loud {
		return Loud
	}
	if c.flags.Verbosity.Quiet {
		return Quiet
	}
	if c.flags.Verbosity.Silent {
		return Silent
	}
	return Normal
}
