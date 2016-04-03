package cli

import (
	"flag"

	"github.com/opentable/sous/util/cmdr"
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

var TopLevelCommands = cmdr.Commands{}

const sousHelp = `
invoke sous

sous is a tool to help speed up the build/test/deploy cycle at your organisation

args: <command>

sous helps by automating the boring bits of the build/test/deploy cycle. It
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

func (*Sous) Help() string { return sousHelp }

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

func (*Sous) Execute(args []string) cmdr.Result {
	err := UsageErrorf("usage: sous [options] command")
	err.Tip = "try `sous help` for a list of commands"
	return err
}

func (Sous) Subcommands() cmdr.Commands {
	return TopLevelCommands
}

func (s *Sous) Verbosity() cmdr.Verbosity {
	if s.flags.Verbosity.Debug {
		return cmdr.Debug
	}
	if s.flags.Verbosity.Loud {
		return cmdr.Loud
	}
	if s.flags.Verbosity.Quiet {
		return cmdr.Quiet
	}
	if s.flags.Verbosity.Silent {
		return cmdr.Silent
	}
	return cmdr.Normal
}
