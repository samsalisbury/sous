package cli

import (
	"io"
	"os"

	"github.com/samsalisbury/semv"
)

// CLI represents the Sous CLI client.
type CLI struct {
	// Version is the version of Sous itself.
	Version semv.Version
	// Verbosity is how much information to print to the user's shell.
	Verbosity Verbosity
	// Output is where the return value of commands that have one gets printed.
	Output io.Writer
	// Graph is the dependency injector used to flesh out command dependencies.
	Graph SousCLIGraph
}

// Invoke tells this CLI to begin processing the command entered by the user.
// This method must exit the process.
func (c CLI) Invoke(args []string) {

	c.Verbosity = Normal
	c.Output = os.Stdout

	if err := c.init(); err != nil {
		c.Exit(err)
	}

	// parse sous flags
	// get command
	// parse command flags
	// invoke command
	// display results

}

// init builds the dependency graph, and injects any relevant values into the
// CLI iteslf.
func (c *CLI) init() (err error) {
	c.buildGraph()
	if err != nil {
		return InternalErrorf(err, "unable to build dependency graph")
	}
	return c.Graph.Inject(c)
}

func (c *CLI) Exit(err error) {
	if err == nil {
		os.Exit(0)
	}
	switch err.(type) {
	default:
		// Wrap this error in an internal error and pass back to this method.
		c.Exit(InternalErrorf(err, "unexpected error"))
	case InternalError:
		os.Exit(EX_SOFTWARE)
	case UsageError:
		os.Exit(EX_USAGE)
	}
}
