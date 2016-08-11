package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/whitespace"
	"github.com/samsalisbury/semv"
)

// Sous is the main sous command.
type Sous struct {
	// CLI is a reference to the CLI singleton. We use it here to set global
	// verbosity.
	CLI *cmdr.CLI
	// Err is the error message stream.
	Err *ErrOut
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

// TopLevelCommands is populated once per command file (beginning sous_) in this
// directory.
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
		"silent: silence all non-essential output")
	fs.BoolVar(&s.flags.Verbosity.Quiet, "q", false,
		"quiet: output only essential error messages")
	fs.BoolVar(&s.flags.Verbosity.Loud, "v", false,
		"loud: output extra info, including all shell commands")
	fs.BoolVar(&s.flags.Verbosity.Debug, "d", false,
		"debug: output detailed logs of internal operations")
}

// RegisterOn adds the Sous object itself to the Psyringe
func (s *Sous) RegisterOn(psy Addable) {
	psy.Add(s)
}

func (s *Sous) Execute(args []string) cmdr.Result {
	r := s.CLI.InvokeWithoutPrinting([]string{"sous", "help"})
	success, ok := r.(cmdr.SuccessResult)
	if !ok {
		return s.usage()
	}
	return UsageErrorf(whitespace.Trim(success.String()) + "\n")
}

func (s *Sous) usage() cmdr.ErrorResult {
	err := UsageErrorf("usage: sous [options] command")
	err.Tip = "try `sous help` for a list of commands"
	return err
}

func (s *Sous) Subcommands() cmdr.Commands {
	//s.CLI.SetVerbosity(s.Verbosity())
	s.Verbosity()
	return TopLevelCommands
}

func (s *Sous) Verbosity() cmdr.Verbosity {
	fmt.Println(s.flags.Verbosity)
	if s.flags.Verbosity.Debug {
		fmt.Println("debug level")
		if s.flags.Verbosity.Loud {
			sous.Log.Vomit.SetOutput(os.Stderr)
		}
		sous.Log.Debug.SetOutput(os.Stderr)
		sous.Log.Info.SetOutput(os.Stderr)

		sous.Log.Vomit.Println("Verbose debugging enabled")
		sous.Log.Debug.Println("Regular debugging enabled")
		sous.Log.Info.Println("Informational messages enabled")
		return cmdr.Debug
	}
	if s.flags.Verbosity.Loud {
		sous.Log.Info.SetOutput(os.Stderr)
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
