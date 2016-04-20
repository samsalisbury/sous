// The cli package implements the Sous Command Line Interface. It is a
// presentation layer, and contains no core logic.
package cli

import (
	"io"
	"os"

	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/yaml"
	"github.com/samsalisbury/semv"
)

// Func aliases, for convenience returning from commands.
var (
	SuccessData       = cmdr.SuccessData
	Successf          = cmdr.Successf
	Success           = cmdr.Success
	UsageErrorf       = cmdr.UsageErrorf
	OSErrorf          = cmdr.OSErrorf
	IOErrorf          = cmdr.IOErrorf
	InternalErrorf    = cmdr.InternalErrorf
	EnsureErrorResult = cmdr.EnsureErrorResult
)

func SuccessYAML(v interface{}) cmdr.Result {
	b, err := yaml.Marshal(v)
	if err != nil {
		return InternalErrorf("unable to marshal YAML: %s", err)
	}
	return SuccessData(b)
}

func NewSousCLI(v semv.Version, out, errout io.Writer) (*cmdr.CLI, error) {

	s := &Sous{Version: v}

	stdout := cmdr.NewOutput(out)
	stderr := cmdr.NewOutput(errout)

	c := &cmdr.CLI{
		Root: s,
		Out:  stdout, Err: stderr,
		// HelpCommand is shown to the user if they type something that looks
		// like they want help, but which isn't recognised by Sous properly. It
		// uses the standard flag.ErrHelp value to decide whether or not to show
		// this.
		HelpCommand: os.Args[0] + " help",
	}

	// Create the CLI dependency graph.
	g, err := BuildGraph(s, c)
	if err != nil {
		return nil, err
	}

	// Before Execute is called on any command, inject it with values from the
	// graph.
	c.Hooks.PreExecute = func(c cmdr.Command) error { return g.Inject(c) }

	return c, nil
}
