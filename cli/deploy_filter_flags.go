package cli

const (
	sourceFlagHelp = `
	-source
		source code location (alternative to -repo and -offset combination)`

	repoFlagHelp = `
	-repo REPOSITORY_NAME
		source code repository location`

	offsetFlagHelp = `
	-offset RELATIVE_PATH
		source code relative repository offset`

	flavorFlagHelp = `
	-flavor FLAVOR
		flavor is a short string used to differentiate alternative deployments`

	tagFlagHelp = `
	-tag TAG_NAME
		source code revision tag`

	revisionFlagHelp = `
	-revision REVISION_ID
		the ID of a revision in the repository to act upon`

	clusterFlagHelp = `
	-cluster CLUSTER
		the deployment environment to consider`

	allFlagHelp = `
	-all
	  all deployments should be considered`
)

var (
	// ClusterFilterFlagsHelp just exposes the -cluster flag (for server)
	ClusterFilterFlagsHelp = clusterFlagHelp
	// ManifestFilterFlagsHelp the text/config for selecting manifests
	ManifestFilterFlagsHelp = sourceFlagHelp + repoFlagHelp + offsetFlagHelp + flavorFlagHelp
	// MetadataFilterFlagsHelp the the text/config for metadata commands
	MetadataFilterFlagsHelp = sourceFlagHelp + repoFlagHelp + offsetFlagHelp + flavorFlagHelp + clusterFlagHelp
	// SourceFlagsHelp is the text (and config) for source flags
	SourceFlagsHelp = repoFlagHelp + offsetFlagHelp + flavorFlagHelp + tagFlagHelp + revisionFlagHelp
	// RectifyFilterFlagsHelp is the text (and config) for rectification flags
	RectifyFilterFlagsHelp = repoFlagHelp + offsetFlagHelp + flavorFlagHelp + clusterFlagHelp + allFlagHelp
	// DeployFilterFlagsHelp is the text and config for deploy flags
	DeployFilterFlagsHelp = repoFlagHelp + offsetFlagHelp + flavorFlagHelp + clusterFlagHelp + allFlagHelp + tagFlagHelp
	// NewDeployFilterFlagsHelp is the text and config for deploy flags
	NewDeployFilterFlagsHelp = repoFlagHelp + offsetFlagHelp + flavorFlagHelp + clusterFlagHelp + tagFlagHelp
)
