package cli

// BuildFlags are CLI flags used to set build options.
type BuildFlags struct {
	Strict  bool   `flag:"strict"`
	Builder string `flag:"builder"`
}

// DeployFlags are CLI flags used to set deployment context and options.
type DeployFlags struct {
	Deployer   string `flag:"deployer"`
	DryRun     bool   `flag:"dry-run"`
	ForceClone bool   `flag:"force-clone"`
	Cluster    string `flag:"cluster"`
}

var deployFlags = `
	-deployment.cluster=CLUSTER_NAME
		set deployment context: cluster

	-deployment.dry-run
		do not make any changes deployments`

var buildFlags = `
	-build.strict
		fail build if any advisories apply

	-build.force-clone
		force building from a new shallow clone of the source context.`
