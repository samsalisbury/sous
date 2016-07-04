package sous

type (
	// Deployer describes a complete deployment system, which is able to create,
	// read, update, and delete deployments.
	Deployer interface {
		RectificationClient
		GetRunningDeployment(fromURLs []string) (Deployments, error)
	}
)
