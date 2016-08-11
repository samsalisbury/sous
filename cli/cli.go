// Package cli implements the Sous Command Line Interface. It is a
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

// Addable objects are able to receive lists of interface{}, presumably to add
// them to a DI registry. Abstracts Psyringe's Add()
type Addable interface {
	Add(...interface{})
}

// A Registrant is able to add values to an Addable (implicitly: a Psyringe)
type Registrant interface {
	RegisterOn(Addable)
}

// SuccessYAML lets you return YAML on the command line.
func SuccessYAML(v interface{}) cmdr.Result {
	b, err := yaml.Marshal(v)
	if err != nil {
		return InternalErrorf("unable to marshal YAML: %s", err)
	}
	return SuccessData(b)
}

// NewSousCLI creates a new Sous cli app.
func NewSousCLI(v semv.Version, out, errout io.Writer) (*cmdr.CLI, error) {

	s := &Sous{Version: v}

	stdout := cmdr.NewOutput(out)
	stderr := cmdr.NewOutput(errout)

	cli := &cmdr.CLI{
		Root: s,
		Out:  stdout, Err: stderr,
		// HelpCommand is shown to the user if they type something that looks
		// like they want help, but which isn't recognised by Sous properly. It
		// uses the standard flag.ErrHelp value to decide whether or not to show
		// this.
		HelpCommand: os.Args[0] + " help",
	}

	var chain []cmdr.Command
	cli.Hooks.Parsed = func(cmd cmdr.Command) error {
		chain = append(chain, cmd)
	}

	// Before Execute is called on any command, inject it with values from the
	// graph.
	cli.Hooks.PreExecute = func(cmd cmdr.Command) error {
		// Create the CLI dependency graph.
		g := BuildGraph(cli)
		for _, c := range chain {
			if r, ok := c.(Registrant); ok {
				r.RegisterOn(g)
			}
		}
		return g.Inject(exe.Cmd)
	}

	return cli, nil
}
