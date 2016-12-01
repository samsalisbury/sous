package main

import (
	"github.com/opentable/sous/util/cmdr"
	"os"
)

var cmds = cmdr.Commands{}
var root = rootCommand{}

func main() {
	so := cmdr.NewOutput(os.Stdout)
	se := cmdr.NewOutput(os.Stderr)

	c := cmdr.CLI{
		Root:        &root,
		Out:         so,
		Err:         se,
		HelpCommand: os.Args[0] + " help",
	}

	cmds["help"] = &helpCommand{}
	cmds["other"] = &otherCommand{}
	cmds["complicated"] = &complicatedCommand{}
	c.Invoke(os.Args)
}
