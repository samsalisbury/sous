// Package cli implements the Sous Command Line Interface. It is a
// presentation layer, and contains no core logic.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
)

// Func aliases, for convenience returning from commands.
var (
	GeneralErrorf = func(format string, a ...interface{}) cmdr.ErrorResult {
		return EnsureErrorResult(fmt.Errorf(format, a...))
	}
	EnsureErrorResult = func(err error) cmdr.ErrorResult {
		sous.Log.Debug.Println(err)
		return cmdr.EnsureErrorResult(err)
	}
)

// ProduceResult converts errors into Results
func ProduceResult(err error) cmdr.Result {
	if err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}

type (
	// CLI describes the command line interface for Sous
	CLI struct {
		*cmdr.CLI
		baseGraph, SousGraph *graph.SousGraph
	}
	// Addable objects are able to receive lists of interface{}, presumably to add
	// them to a DI registry. Abstracts Psyringe's Add()
	Addable interface {
		Add(...interface{})
	}

	// A Registrant is able to add values to an Addable (implicitly: a Psyringe)
	Registrant interface {
		RegisterOn(Addable)
	}
)

// SuccessYAML lets you return YAML on the command line.
func SuccessYAML(v interface{}) cmdr.Result {
	b, err := yaml.Marshal(v)
	if err != nil {
		return cmdr.InternalErrorf("unable to marshal YAML: %s", err)
	}
	return cmdr.SuccessData(b)
}

// Plumbing injects a command with the current psyringe,
// then it Executes it, returning the result.
func (cli *CLI) Plumbing(cmd cmdr.Executor, args []string) cmdr.Result {
	if err := cli.Plumb(cmd); err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	return cmd.Execute(args)
}

// Plumb injects a lists of commands with the currect psyringe, returning early in the event of an error
func (cli *CLI) Plumb(cmds ...cmdr.Executor) error {
	for _, cmd := range cmds {
		if err := cli.SousGraph.Inject(cmd); err != nil {
			return err
		}
	}
	return nil
}

// BuildCLIGraph builds the CLI DI graph.
func BuildCLIGraph(cli *CLI, root *Sous, in io.Reader, out, err io.Writer) *graph.SousGraph {
	g := cli.baseGraph.Clone()
	g.Add(cli)
	g.Add(root)
	g.Add(func(c *CLI) graph.Out {
		return graph.Out{Output: c.Out}
	})
	g.Add(func(c *CLI) graph.ErrOut {
		return graph.ErrOut{Output: c.Err}
	})
	cli.SousGraph = &graph.SousGraph{Psyringe: g} //Ugh, weird state.

	return cli.SousGraph
}

// NewSousCLI creates a new Sous cli app.
func NewSousCLI(s *Sous, in io.Reader, out, errout io.Writer) (*CLI, error) {

	stdout := cmdr.NewOutput(out)
	stderr := cmdr.NewOutput(errout)

	cli := &CLI{
		CLI: &cmdr.CLI{
			Root: s,
			Out:  stdout,
			Err:  stderr,
			// HelpCommand is shown to the user if they type something that looks
			// like they want help, but which isn't recognised by Sous properly. It
			// uses the standard flag.ErrHelp value to decide whether or not to show
			// this.
			HelpCommand: os.Args[0] + " help",
		},

		baseGraph: graph.BuildGraph(in, out, errout),
	}

	var g *graph.SousGraph
	var chain []cmdr.Command

	cli.Hooks.Startup = func(*cmdr.CLI) error {
		g = BuildCLIGraph(cli, s, in, out, errout)
		chain = make([]cmdr.Command, 0)
		return nil
	}

	cli.Hooks.Parsed = func(cmd cmdr.Command) error {
		chain = append(chain, cmd)
		return nil
	}

	// Before Execute is called on any command, inject it with values from the
	// graph.
	cli.Hooks.PreExecute = func(cmd cmdr.Command) error {
		// Create the CLI dependency graph.

		for _, c := range chain {
			if r, ok := c.(Registrant); ok {
				r.RegisterOn(g)
			}
		}

		// This is tricky: we need to inject the completely added graph to the
		// graph itself so that sous_server can pass a complete graph to
		// server.RunServer
		g.Add(g)

		for _, c := range chain {
			if err := g.Inject(c); err != nil {
				return err
			}
		}
		return nil
	}

	cli.Hooks.PreFail = func(err error) cmdr.ErrorResult {
		if err != nil {
			originalErr := fmt.Sprint(err)
			err = errors.Cause(err)
			causeStr := err.Error()
			if originalErr != causeStr {
				sous.Log.Debug.Println(originalErr)
			}
		}
		return EnsureErrorResult(err)
	}

	return cli, nil
}
