package sous

type (
	State struct {
		Applications Applications
		Buildpacks   Buildpacks
		Contracts    Contracts
		Deployments  Deployments
		Clusters     Clusters
	}
	Buildpacks  []Buildpack
	Contracts   []Contract
	Deployments []Deployment

	Buildpack  struct{}
	Contract   struct{}
	Deployment struct{}
)
