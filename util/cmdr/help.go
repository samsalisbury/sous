package cmdr

import (
	"bytes"
	"flag"
	"fmt"
	"strings"

	"github.com/opentable/sous/util/whitespace"
)

// Help contains help strings describing a command.
type Help struct{ Short, Desc, Args, Long string }

// ParseHelp parses a command's help string into its component parts.
func ParseHelp(s string) *Help {
	chunks := strings.SplitN(s, "\n\n", 4)
	pieces := []string{}
	for _, c := range chunks {
		c = whitespace.Trim(c)
		if len(s) != 0 {
			pieces = append(pieces, c)
		}
	}
	h := &Help{
		"error: no short description defined",
		"error: no description defined",
		"",
		"error: no help text defined",
	}
	if len(pieces) > 0 {
		h.Short = pieces[0]
	}
	if len(pieces) > 1 {
		h.Desc = pieces[1]
	}
	if len(pieces) > 2 {
		h.Args = whitespace.Trim(strings.TrimPrefix(pieces[2], "args:"))
	}
	if len(pieces) == 3 {
		h.Long = pieces[2]
	}
	return h
}

// Usage prints the usage message.
func (h *Help) Usage(name string) string {
	return fmt.Sprintf("usage: %s %s", name, h.Args)
}

// PrintHelp recursively descends down the commands and subcommands named in its
// arguments, and prints the help for the deepest member it meets, or returns an
// error if no such command exists.
func (cli *CLI) PrintHelp(base Command, name string, args []string) error {
	return cli.printHelp(cli.Out, base, name, args)
}

// Help is similar to PrintHelp, except it returns the result as a string
// instead of writing to the CLI's default Output.
func (cli *CLI) Help(base Command, name string, args []string) (string, error) {
	b := &bytes.Buffer{}
	err := cli.printHelp(NewOutput(b), base, name, args)
	return b.String(), err
}

func (cli *CLI) printHelp(out *Output, base Command, name string, args []string) error {
	if len(args) == 0 {
		help := ParseHelp(base.Help())
		out.Println(help.Usage(name))
		out.Println()
		out.Println(help.Desc)
		cli.printSubcommands(out, base, name)
		cli.printOptions(out, base, name)
		return nil
	}
	hasSubCommands, ok := base.(Subcommander)
	if !ok {
		return UsageErrorf("%q does not have any subcommands", name)
	}
	scs := hasSubCommands.Subcommands()
	subcommandName := args[0]
	name = name + " " + subcommandName
	sc, ok := scs[subcommandName]
	if !ok {
		return UsageErrorf("command %q does not exist", name)
	}
	args = args[1:]
	return cli.printHelp(out, sc, name, args)
}

func (cli *CLI) printSubcommands(out *Output, c Command, name string) {
	subcommander, ok := c.(Subcommander)
	if !ok {
		return
	}
	cs := subcommander.Subcommands()
	out.Println("\nsubcommands:")
	out.Indent()
	defer out.Outdent()
	out.Table(commandTable(cs))
}

func (cli *CLI) printOptions(out *Output, command Command, name string) {
	addsFlags, ok := command.(AddsFlags)
	if !ok {
		return
	}
	out.Println("\noptions:")
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	addsFlags.AddFlags(fs)
	fs.SetOutput(out)
	fs.PrintDefaults()
}

func commandTable(cs Commands) [][]string {
	t := make([][]string, len(cs))
	for i, name := range cs.SortedKeys() {
		t[i] = make([]string, 2)
		t[i][0] = name
		t[i][1] = ParseHelp(cs[name].Help()).Short
	}
	return t
}
