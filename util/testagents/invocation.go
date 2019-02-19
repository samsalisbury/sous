package testagents

import "fmt"

// invocation is the invocation directly from the test, without any formatting
// or manipulation.
type invocation struct {
	name, subcmd string
	flags        Flags
	args         []string
	finalArgs    []string
}

// String returns this invocation roughly as a copy-pastable shell command.
// Note: if args contain quotes some manual editing may be required.
func (i invocation) String() string {
	return fmt.Sprintf("%s %s", i.name, quotedArgsString(i.finalArgs))
}
