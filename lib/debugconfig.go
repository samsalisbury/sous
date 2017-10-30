package sous

import "github.com/davecgh/go-spew/spew"

// On init, set up some spew defaults so that debugging and logging will be a
// trifle more useful.
func init() {
	// prevent String() or Error() from concealing contents of structs
	spew.Config.ContinueOnMethod = true
	// sort keys in maps for consistent output
	spew.Config.SortKeys = true
}
