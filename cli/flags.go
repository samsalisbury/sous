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
		do not make any changes deployments
		
		Instead of performing deployment actions on running clusters, just
		display the actions that would be taken without this flag.
`

var buildFlags = `
	-build.strict
		fail build if any advisories apply

		Intended to produce production-grade build artifacts.

	-build.force-clone
		build from a fresh isolated clone

		Force building from a new shallow clone of the source context.

		Usually if building inside a repository using the default context, the
		code inside the current working directory is built. If you specify a
		context which does not match the default context, the source identified
		the context is cloned into your local scratch directory, and built from
		there.
`
