package cmdr

import (
	"bytes"
	"flag"
	"fmt"
	"strings"
)

// Help collects information about a subcommand and its arguments, descends
// the path down the command tree provided by cmdArgs, finds the lowest
// subcommand on that path, and returns the help text for that subcommand.
func (cli *CLI) Help(cmd Command, cmdArgs []string) (string, error) {
	b := &bytes.Buffer{}
	bottomSubcmd := findBottomCommand(cmd, cmdArgs)
	err := cli.printFullHelp(NewOutput(b), *bottomSubcmd)
	return b.String(), err
}

func (cli *CLI) printFullHelp(out *Output, cmd Command) error {
	help := cmd.Help()
	if len(help) == 0 {
		return fmt.Errorf("No help available for command")
	}
	out.Println(cmd.Help())
	cli.printSubcommands(out, cmd)
	cli.printOptions(out, cmd)
	return nil
}

func (cli *CLI) printSubcommands(out *Output, c Command) {
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

func (cli *CLI) printOptions(out *Output, command Command) {
	addsFlags, ok := command.(AddsFlags)
	if !ok {
		return
	}
	out.Println("\noptions:")
	fs := flag.NewFlagSet("help", flag.ContinueOnError)
	addsFlags.AddFlags(fs)
	fs.SetOutput(out)
	fs.PrintDefaults()
}

func commandTable(cs Commands) [][]string {
	t := make([][]string, len(cs))
	for i, name := range cs.SortedKeys() {
		var shortHelp string
		splitHelp := strings.Split(cs[name].Help(), "\n")
		if len(splitHelp) > 0 {
			shortHelp = splitHelp[0]
		}
		t[i] = make([]string, 2)
		t[i][0] = name
		t[i][1] = shortHelp
	}
	return t
}

// findBottomCommand exists to satisfy this rule: "The arguments to a command
// can either be values or indicative of a subcommand." It traverses the list
// of command arguments to find the subcommand furthest down the tree.
func findBottomCommand(cmd Command, cmdArgs []string) *Command {
	bottomSubCmd := &cmd
	for _, a := range cmdArgs {
		// check if the command has any subcommands
		testCmd := *bottomSubCmd
		hasSubCmd, ok := testCmd.(Subcommander)
		if !ok {
			return bottomSubCmd
		}
		childCmd, ok := hasSubCmd.Subcommands()[a]
		if !ok {
			return bottomSubCmd
		}
		bottomSubCmd = &childCmd
	}
	return bottomSubCmd
}
