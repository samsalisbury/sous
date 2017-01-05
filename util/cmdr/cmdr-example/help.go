package main

import (
	"github.com/opentable/sous/util/cmdr"
)

type helpCommand struct {
	CLI *cmdr.CLI
}

func (c helpCommand) Execute(subcommands []string) cmdr.Result {
	help, err := c.CLI.Help(root, subcommands)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	return cmdr.Successf(help)
}

func (c helpCommand) Help() string {
	return "\n" + "help provides help." + "\n" + "\n"
}
