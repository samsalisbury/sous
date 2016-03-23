package cli

import (
	"flag"
	"os"

	"github.com/opentable/sous/util/cmdr"
)

type SousHelp struct {
	Out  Out
	Sous *Sous
}

const sousHelpHelp = `
get help with sous

help shows help information for sous itself, as well as all its subcommands
for detailed help with any command, use 'sous help <command>'.

args: [command]
`

func (sh *SousHelp) Help() *cmdr.Help { return cmdr.ParseHelp(sousHelpHelp) }

func (sh *SousHelp) Execute(args []string) cmdr.Result {
	// Get the name this instance was invoked with.
	name := os.Args[0]
	if err := sh.printHelp(args, name, sh.Sous); err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	return cmdr.Successf("\nsous version %s", sh.Sous.Version)
}

// printHelp recursively descends down the commands and subcommands named in its
// arguments, and prints the help for the deepest member it meets, or returns an
// error if no such command exists.
func (sh *SousHelp) printHelp(args []string, name string, c cmdr.Command) error {
	if len(args) == 0 {
		help := c.Help()
		sh.Out.Println(help.Usage(name))
		sh.Out.Println()
		sh.Out.Println(help.Desc)
		sh.printSubcommands(name, c)
		sh.printOptions(name, c)
		return nil
	}
	hasSubCommands, ok := c.(cmdr.Subcommander)
	if !ok {
		return cmdr.UsageErrorf("%q does not have any subcommands", name)
	}
	scs := hasSubCommands.Subcommands()
	subcommandName := args[0]
	name = name + " " + subcommandName
	sc, ok := scs[subcommandName]
	if !ok {
		return cmdr.UsageErrorf("command %q does not exist", name)
	}
	args = args[1:]
	return sh.printHelp(args, name, sc)
}

func (sh *SousHelp) printSubcommands(name string, c cmdr.Command) {
	subcommander, ok := c.(cmdr.Subcommander)
	if !ok {
		return
	}
	cs := subcommander.Subcommands()
	sh.Out.Println("\nsubcommands:")
	sh.Out.Indent()
	defer sh.Out.Outdent()
	sh.Out.Table(commandTable(cs))
}

func (sh *SousHelp) printOptions(name string, command cmdr.Command) {
	addsFlags, ok := command.(cmdr.AddsFlags)
	if !ok {
		return
	}
	sh.Out.Println("\noptions:")
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	addsFlags.AddFlags(fs)
	fs.SetOutput(sh.Out)
	fs.PrintDefaults()
}

func commandTable(cs cmdr.Commands) [][]string {
	t := make([][]string, len(cs))
	for i, name := range cs.SortedKeys() {
		t[i] = make([]string, 2)
		t[i][0] = name
		t[i][1] = cs[name].Help().Short
	}
	return t
}
