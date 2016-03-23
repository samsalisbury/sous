package cmdr

import (
	"fmt"
	"strings"

	"github.com/opentable/sous/util/whitespace"
)

type (
	Help struct{ Short, Desc, Args, Long string }
)

func ParseHelp(s string) *Help {
	chunks := strings.SplitN(s, "\n\n", 4)
	pieces := []string{}
	for _, c := range chunks {
		c = whitespace.Trim(c)
		if len(s) != 0 {
			pieces = append(pieces, c)
		}
	}
	hc := &Help{
		"error: no short description defined",
		"error: no description defined",
		"",
		"error: no help text defined",
	}
	if len(pieces) > 0 {
		hc.Short = pieces[0]
	}
	if len(pieces) > 1 {
		hc.Desc = pieces[1]
	}
	if len(pieces) > 2 {
		hc.Args = whitespace.Trim(strings.TrimPrefix(pieces[2], "args:"))
	}
	if len(pieces) == 3 {
		hc.Long = pieces[2]
	}
	return hc
}

func (c *Help) Usage(name string) string {
	return fmt.Sprintf("usage: %s %s", name, c.Args)
}
