package cli

// Verbosity represents the level of output detail a CLI should give to the
// user.
type Verbosity string

const (
	// Silent means output absolutely no error or warning messsages, but still
	// output the result of a command, if it has a real result. For example
	// `sous image` whose primary function is to must echo the name of a docker
	// image. Note that if the command fails, you will have only the exit code
	// to go by, and no error messages.
	Silent = Verbosity("silent")
	// Quiet is similar to silent, but will echo error messages if the command
	// cannot be completed successfully.
	Quiet = Verbosity("quiet")
	// Normal is the default verbosity, and is similar to quiet, but will
	// additionally output tips and warnings to the user. For long-running
	// commands, Normal may additionally output status updates, progress meters,
	// and other information to let the user know it's still working.
	Normal = Verbosity("normal")
	// Loud is similar to Normal, but outputs additional status information,
	// and the full text of all shell commands issued by sous.
	Loud = Verbosity("loud")
	// Debug is similar to Loud, but additionally outputs detailed internal
	// operations, helpful in debugging problems.
	Debug = Verbosity("debug")
)
