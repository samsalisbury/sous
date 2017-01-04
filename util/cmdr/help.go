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
	// discarding the index to the bottom command in the cmdArgs slice,
	// not discarding an error.
	bottomSubcmd, _ := findBottomCommand(cmd, cmdArgs)
	return cli.formatFullHelp(*bottomSubcmd)
}

func (cli *CLI) formatFullHelp(cmd Command) (string, error) {
	b := &bytes.Buffer{}
	help := cmd.Help()
	if len(help) == 0 {
		msg := "no help available for command"
		return msg, fmt.Errorf(msg)
	}
	b.WriteString(cmd.Help())
	b.WriteString(formatSubcommands(cmd))
	b.WriteString(cli.formatFlags(cmd))
	return b.String(), nil
}

func formatSubcommands(c Command) string {
	b := &bytes.Buffer{}
	out := NewOutput(b)
	subcommander, ok := c.(Subcommander)
	if !ok {
		return ""
	}
	cs := subcommander.Subcommands()
	out.Println("\n\nsubcommands:\n")
	out.Indent()
	defer out.Outdent()
	out.Table(commandTable(cs))
	return b.String()
}

func (cli *CLI) formatFlags(command Command) string {
	addsFlags, ok := command.(AddsFlags)
	if !ok {
		return ""
	}
	b := &bytes.Buffer{}
	b.WriteString("\n\noptions:\n")
	fs := flag.NewFlagSet("help", flag.ContinueOnError)
	addsFlags.AddFlags(fs)
	fs.SetOutput(b)
	fs.PrintDefaults()
	return b.String()
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
func findBottomCommand(cmd Command, cmdArgs []string) (*Command, int) {
	var pos int
	bottomSubCmd := &cmd
	for pos = 0; pos < len(cmdArgs); pos++ {
		// check if the command has any subcommands
		testCmd := *bottomSubCmd
		withSubCmd, ok := testCmd.(Subcommander)
		if !ok {
			// there are no more subcommands, this loop is done.
			return bottomSubCmd, pos
		}

		nextToken := cmdArgs[pos]
		childCmd, ok := withSubCmd.Subcommands()[nextToken]
		if !ok {
			// this command has subcommands, but this argument isn't one of them.
			return bottomSubCmd, pos
		}
		bottomSubCmd = &childCmd
	}
	return bottomSubCmd, pos
}
