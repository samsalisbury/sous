package main

import (
	"bytes"
	"github.com/opentable/sous/util/cmdr"
)

type complicatedCommand struct{}

var complicatedSubcommands = cmdr.Commands{}

func (c complicatedCommand) Help() string {
	return "This is a complicated subcommand.\n"
}

func (c complicatedCommand) Execute(args []string) cmdr.Result {
	b := bytes.Buffer{}
	b.WriteString(c.Help())
	b.WriteString("\n")
	if len(complicatedSubcommands) > 0 {
		b.WriteString("subcommands:\n")
		b.WriteString(subTable(c.Subcommands()))
	}
	return cmdr.Successf(b.String())
}

func (c complicatedCommand) Subcommands() cmdr.Commands {
	return complicatedSubcommands
}
