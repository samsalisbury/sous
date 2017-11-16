package cli

import (
	"flag"

	"github.com/opentable/sous/config"
)

// AddVerbosityFlags adds the -s -q -v -d flags to fs, linking them to the
// provided config.Verbosity pointer v.
func AddVerbosityFlags(v *config.Verbosity) func(*flag.FlagSet) {
	return func(fs *flag.FlagSet) {
		fs.BoolVar(&v.Silent, "s", false,
			"silent: silence all non-essential output")
		fs.BoolVar(&v.Quiet, "q", false,
			"quiet: output only essential error messages")
		fs.BoolVar(&v.Loud, "v", false,
			"loud: output extra info, including all shell commands")
		fs.BoolVar(&v.Debug, "d", false,
			"debug: output detailed logs of internal operations")
	}
}
