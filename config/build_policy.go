package config

type (
	// PolicyFlags capture user intent about the processing of a build
	PolicyFlags struct {
		ForceClone, Strict bool
	}
)
