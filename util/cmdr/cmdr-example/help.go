package main

import (
	"fmt"
	"github.com/opentable/sous/util/cmdr"
)

type helpCommand struct{}

func (c helpCommand) Execute(args []string) cmdr.Result {
	helpMessage := "cmdr-example hello world\n\n"
	if len(args) > 0 {
		// the subcommand is the first string in args
		cName := args[0]
		// check that the Command struct exists within the map of
		// Command structs.
		cmd, ok := cmds[cName]
		if !ok {
			// There wasn't a Command struct representing the requested
			// command stored in the map of Command structs, so we're
			// letting the user know that there isn't any help available.
			helpMessage = helpMessage + fmt.Sprintf("No help for command %s.", cName)
		} else {
			helpMessage = helpMessage + cmd.Help()
		}
	} else {
		helpMessage = helpMessage + "subcommands:\n\n"
		helpMessage = helpMessage + subTable(root.Subcommands())
	}
	return cmdr.Successf(helpMessage)
}

func (c helpCommand) Help() string {
	return "help provides help."
}
