package main

import (
	"bytes"
	"fmt"

	"github.com/opentable/sous/util/cmdr"
)

func subTable(cmds cmdr.Commands) string {
	b := bytes.Buffer{}
	for c, s := range cmds {
		b.WriteString(fmt.Sprintf("%15s: %s\n", c, s.Help()))
	}
	return b.String()
}
