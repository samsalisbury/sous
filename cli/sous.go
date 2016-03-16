package cli

import (
	"flag"
	"io"
	"os"

	"github.com/samsalisbury/semv"
)

// CLI represents the Sous CLI client. It is the root command, and so implements
// Command.
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
	c.Output = os.Stdout
	if err := c.init(); err != nil {
		return ErrorResult(err)
	}
	return nil
}

// init builds the dependency graph, and injects any relevant values into the
// CLI iteslf.
func (c *Sous) init() error {
	if err := c.buildGraph(); err != nil {
		return InternalError{
			Err:     err,
			Message: "unable to build dependency graph",
		}
	}
	return c.Graph.Inject(c)
}

func (c *Sous) Success() {
	os.Exit(0)
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
