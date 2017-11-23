package config

import "github.com/opentable/sous/util/logging"

// Verbosity configures how chatty Sous is on its logs
type Verbosity struct {
	Silent, Quiet, Loud, Debug bool
}

// UpdateLevel updates the logging level of the provided LogSet
// to reflect this Verbosity
func (v Verbosity) UpdateLevel(set *logging.LogSet) {
	switch {
	default:
		// do nothing - if no flag provided, keep configured levels
	case v.Silent:
		set.BeSilent()
	case v.Quiet:
		set.BeQuiet()
	case v.Loud:
		set.BeChatty()
	case v.Debug:
		set.BeHelpful()
	}
}
