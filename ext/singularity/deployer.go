package singularity

type (
	// Deployer implements the sous.Deployer interface by stitching together
	// RectiAgent and SetCollector.
	Deployer struct {
		*RectiAgent
		*SetCollector
	}
)
