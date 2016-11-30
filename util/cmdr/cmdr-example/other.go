package main

import (
	"github.com/opentable/sous/util/cmdr"
)

type otherCommand struct{}

func (c otherCommand) Help() string {
	return "other help"
}

func (c otherCommand) Execute(args []string) cmdr.Result {
	return cmdr.Successf("other executes!")
}
