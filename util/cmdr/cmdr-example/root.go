package main

import (
	"github.com/opentable/sous/util/cmdr"
)

type rootCommand struct{}

func (c rootCommand) Subcommands() cmdr.Commands {
	return cmds
}

func (c rootCommand) Help() string {
	return ("Use 'cmdr-example help' for a list of commands")
}

func (c rootCommand) Execute(args []string) cmdr.Result {
	return cmdr.Successf(c.Help())
}
