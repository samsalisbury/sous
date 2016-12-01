package main

import "github.com/opentable/sous/util/cmdr"

type fakeComplicatedCommand struct{}

func init() {
	complicatedSubcommands["fake"] = &fakeComplicatedCommand{}
}

func (c fakeComplicatedCommand) Help() string {
	return "Fakery."
}

func (c fakeComplicatedCommand) Execute() cmdr.Result {
	return cmdr.Successf("Complicated and fake.")
}
