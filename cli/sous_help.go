package cli

import (
	"flag"
	"os"
)

type SousHelp struct {
	Out
	Sous *Sous
}

const sousHelpHelp = `
get help with sous

help shows help information for sous itself, as well as all its subcommands
for detailed help with any command, use 'sous help <command>'.

args: [command]
`

func (sh *SousHelp) Help() *Help { return ParseHelp(sousHelpHelp) }

func (sh *SousHelp) Execute(args []string) Result {
	// Get the name this instance was invoked with.
	name := os.Args[0]
	sh.printHelp(args, name, sh.Sous)
	return Successf("\nsous version %s", sh.Sous.Version)
}

// printHelp recursively descends down the commands and subcommands named in its
// arguments, and prints the help for the deepest member it meets, or returns an
// error if no such command exists.
func (sh *SousHelp) printHelp(args []string, name string, c Command) error {
	if len(args) == 0 {
		help := c.Help()
		sh.Println(help.Usage(name))
		sh.Println()
		sh.Println(help.Desc)
		sh.printSubcommands(name, c)
		sh.printOptions(name, c)
		return nil
	}
	hasSubCommands, ok := c.(Subcommander)
	if !ok {
		return UsageErrorf(nil, "%s does not have any subcommands")
	}
	scs := hasSubCommands.Subcommands()
	subcommandName := args[0]
	name = name + " " + subcommandName
	sc, ok := scs[subcommandName]
	if !ok {
		return UsageErrorf(nil, "command %q does not exist", name)
	}
	args = args[1:]
	return sh.printHelp(args, name, sc)
}

func (sh *SousHelp) printSubcommands(name string, c Command) {
	subcommander, ok := c.(Subcommander)
	if !ok {
		return
	}
	cs := subcommander.Subcommands()
	sh.Println("\nsubcommands:")
	sh.Indent()
	defer sh.Outdent()
	sh.Table(commandTable(cs))
}

func (sh *SousHelp) printOptions(name string, command Command) {
	addsFlags, ok := command.(AddsFlags)
	if !ok {
		return
	}
	sh.Println("\noptions:")
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	addsFlags.AddFlags(fs)
	fs.SetOutput(sh.writer)
	fs.PrintDefaults()
}

func commandTable(cs Commands) [][]string {
	t := make([][]string, len(cs))
	for i, name := range cs.SortedKeys() {
		t[i] = make([]string, 2)
		t[i][0] = name
		t[i][1] = cs[name].Help().Short
	}
	return t
}
