package cli

import (
	"flag"

	"github.com/samsalisbury/semv"
)

// Sous is the main sous command.
type Sous struct {
	// Version is the version of Sous itself.
	Version semv.Version
	// flags holds the values of flags passed to this command
	flags struct {
		Help      bool
		Verbosity struct {
			Silent, Quiet, Loud, Debug bool
		}
	}
}

const sousHelp = `
the main sous command

args: <command>

sous is a tool for automating the boring bits of the build/test/deploy cycle. It
provides commands in this CLI for performing all the actions the sous server is
capable of, like building container images, testing them, and instigating
deployments.

sous also has some extra convenience commands for doing things like getting free
ports and host names, managing its own configuration, and spinning up
subsections of your production environment locally, for easy testing.

For a list of commands, use 'sous help'

Please report any issue with sous to https://github.com/opentable/sous/issues
pull requests are welcome.
`

func (*Sous) Help() *Help { return ParseHelp(sousHelp) }

func (s *Sous) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&s.flags.Verbosity.Silent, "s", false,
		"silent verbosity: silence all nonessential output")
	fs.BoolVar(&s.flags.Verbosity.Quiet, "q", false,
		"quiet verbosity: output only essential error messages")
	fs.BoolVar(&s.flags.Verbosity.Loud, "v", false,
		"loud verbosity: output extra info, including all shell commands")
	fs.BoolVar(&s.flags.Verbosity.Debug, "d", false,
		"debug level verbosity: output detailed logs of internal operations")
}

func (*Sous) Execute(args []string, out, errout Output) Result {
	return UsageError{
		Message: "usage: sous [options] command",
		Tip:     "try `sous help` for a list of commands",
	}
}

func (Sous) Subcommands() Commands {
	return Commands{
		"help":    &SousHelp{},
		"version": &SousVersion{},
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
