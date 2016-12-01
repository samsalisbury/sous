package main

import (
	"bytes"
	"fmt"
	"github.com/opentable/sous/util/cmdr"
)

type helpCommand struct{}

func (c helpCommand) Execute(args []string) cmdr.Result {
	b := bytes.Buffer{}
	b.WriteString("cmdr-example hello world\n\n")
	if len(args) > 0 {
		// the subcommand is the first string in args
		cName := args[0]
		b.WriteString(cName + "\n\n")
		// check that the Command struct exists within the map of
		// Command structs.
		cmd, ok := cmds[cName]
		if !ok {
			// There wasn't a Command struct representing the requested
			// command stored in the map of Command structs, so we're
			// letting the user know that there isn't any help available.
			b.WriteString(fmt.Sprintf("No help for command %s.", cName))
		} else {
			b.WriteString(cmd.Help())
		}
	} else {
		b.WriteString("subcommands:\n\n")
		b.WriteString(subTable(root.Subcommands()))
	}
	return cmdr.Successf(b.String())
}

func (c helpCommand) Help() string {
	return "help provides help."
}
