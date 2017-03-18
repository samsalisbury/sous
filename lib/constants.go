package sous

// SingularityDeployMetadataClusterName defines the namespace for storing a Sous ClusterName in SingularityDeploy metadata.
const SingularityDeployMetadataClusterName = "com.opentable.sous.clustername"

// SingularityDeployMetadataFlavor defines the namespace for storing a Sous Flavor in SingularityDeploy metadata.
const SingularityDeployMetadataFlavor = "com.opentable.sous.flavor"

// SingularityDeployTimeout sets the number of seconds to wait for a SingularityDeploy before it is marked failed.
// Increasing this number from the stock 120sec is helpful when dealing with a slow connection to a Docker registry.
const SingularityDeployTimeout = 10 * 60
