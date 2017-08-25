// Package cli implements the Sous Command Line Interface. It is a
// presentation layer, and contains no core logic.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
)

// Func aliases, for convenience returning from commands.
var (
	GeneralErrorf = func(format string, a ...interface{}) cmdr.ErrorResult {
		return EnsureErrorResult(fmt.Errorf(format, a...))
	}
	EnsureErrorResult = func(err error) cmdr.ErrorResult {
		logging.Log.Debug.Println(err)
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
		scopedGraphs         map[cmdr.Command]*graph.SousGraph
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
func (cli *CLI) Plumbing(from cmdr.Command, cmd cmdr.Executor, args []string) cmdr.Result {
	if err := cli.Plumb(from, cmd); err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	return cmd.Execute(args)
}

// Plumb injects a lists of commands with the currect psyringe, returning early in the event of an error
func (cli *CLI) Plumb(from cmdr.Command, cmds ...cmdr.Executor) error {
	for _, cmd := range cmds {
		if err := cli.scopedGraph(from, nil).Inject(cmd); err != nil {
			return err
		}
	}
	return nil
}

// BuildCLIGraph builds the CLI DI graph.
func BuildCLIGraph(cli *CLI, root *Sous, in io.Reader, out, err io.Writer) *graph.SousGraph {
	g := cli.baseGraph //was .Clone() - caused problems
	g.Add(cli)
	g.Add(root)
	g.Add(func(c *CLI) graph.Out {
		return graph.Out{Output: c.Out}
	})
	g.Add(func(c *CLI) graph.ErrOut {
		return graph.ErrOut{Output: c.Err}
	})
	cli.SousGraph = g

	return cli.SousGraph
}

func (cli *CLI) scopedGraph(cmd, under cmdr.Command) *graph.SousGraph {
	if g, ok := cli.scopedGraphs[cmd]; ok {
		return g
	}

	parent := cli.scopedGraphs[under]

	g := &graph.SousGraph{Psyringe: parent.Clone()}
	if r, ok := cmd.(Registrant); ok {
		r.RegisterOn(g)
	}
	cli.scopedGraphs[cmd] = g
	return g
}

// NewSousCLI creates a new Sous cli app.
func NewSousCLI(di *graph.SousGraph, s *Sous, in io.Reader, out, errout io.Writer) (*CLI, error) {

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

		baseGraph: di,
	}

	chain := []cmdr.Command{}
	rootGraph := BuildCLIGraph(cli, s, in, out, errout)
	s.RegisterOn(rootGraph)
	cli.scopedGraphs = map[cmdr.Command]*graph.SousGraph{s: rootGraph}

	cli.Hooks.Parsed = func(cmd cmdr.Command) error {
		chain = append(chain, cmd)
		return nil
	}

	// Before Execute is called on any command, inject it with values from the
	// graph.
	cli.Hooks.PreExecute = func(cmd cmdr.Command) error {
		// Create the CLI dependency graph.

		for n, c := range chain {
			var under cmdr.Command
			if n > 0 {
				under = chain[n-1]
			}
			g := cli.scopedGraph(c, under)
			if err := g.Inject(c); err != nil {
				return errors.Wrapf(err, "setup for execute")
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
				logging.Log.Debugf("%v\n", originalErr)
			}
		}
		return EnsureErrorResult(err)
	}

	return cli, nil
}
