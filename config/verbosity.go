package config

// Verbosity configures how chatty Sous is on its logs
type Verbosity struct {
	Silent, Quiet, Loud, Debug bool
}
