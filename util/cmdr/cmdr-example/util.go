package main

import (
	"bytes"
	"fmt"

	"github.com/opentable/sous/util/cmdr"
)

func subTable(cmds cmdr.Commands) string {
	b := bytes.Buffer{}
	for c, s := range cmds {
		b.WriteString(fmt.Sprintf("\t%s:\t%s\n", c, s.Help()))
	}
	return b.String()
}
