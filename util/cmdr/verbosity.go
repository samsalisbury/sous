package cmdr

// Verbosity represents the level of output detail a CLI should give to the
// user.
type Verbosity int

const (
	_ = iota
	// Silent means output absolutely no error or warning messsages, but still
	// output the result of a command, if it has a real result.
	Silent Verbosity = iota
	// Quiet is similar to silent, but will echo error messages if the command
	// cannot be completed successfully.
	Quiet
	// Normal is the default verbosity, and is similar to quiet, but will
	// additionally output tips and warnings to the user. For long-running
	// commands, Normal may additionally output status updates, progress meters,
	// and other information to let the user know it's still working.
	Normal
	// Loud is similar to Normal, but outputs additional information.
	Loud
	// Debug is similar to Loud, but additionally outputs detailed internal
	// operations, helpful in debugging problems.
	Debug
)

func (v Verbosity) String() string {
	switch v {
	default:
		return "invalid"
	case Silent:
		return "silent"
	case Quiet:
		return "quiet"
	case Normal:
		return "normal"
	case Loud:
		return "loud"
	case Debug:
		return "debug"
	}
}
