package sous

// SingularityDeployTimeout sets the number of seconds to wait for a SingularityDeploy before it is marked failed.
// Increasing this number from the stock 120sec is helpful when dealing with a slow connection to a Docker registry.
const SingularityDeployTimeout = 10 * 60

// SingularityDeployMetadataClusterName defines the namespace for storing a Sous ClusterName in SingularityDeploy metadata.
const ClusterNameLabel = "com.opentable.sous.clustername"

// SingularityDeployMetadataFlavor defines the namespace for storing a Sous Flavor in SingularityDeploy metadata.
const FlavorLabel = "com.opentable.sous.flavor"

// RepoLabel is the metadata fieldname that records the version control repository URL of a Sous-controlled service.
const RepoLabel = "com.opentable.sous.repo_url"

// PathLabel is the metadata fieldname that records the subdirectory under a RepoLabel of a Sous-controlled service.
const PathLabel = "com.opentable.sous.repo_offset"

// VersionLabel is a metadata fieldname that records the semver tag of a Sous-controlled service.
const VersionLabel = "com.opentable.sous.version"

// RevisionLabel is a metadata fieldname that records the git revision ID of a Sous-controlled service.
const RevisionLabel = "com.opentable.sous.revision"
