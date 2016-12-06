package main

import (
	"github.com/opentable/sous/util/cmdr"
)

type complicatedCommand struct{}

var complicatedSubcommands = cmdr.Commands{}

func (c complicatedCommand) Help() string {
	return "This is a complicated subcommand.\n"
}

func (c complicatedCommand) Execute(args []string) cmdr.Result {
	return cmdr.Successf("complicated executes")
}

func (c complicatedCommand) Subcommands() cmdr.Commands {
	return complicatedSubcommands
}
