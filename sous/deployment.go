package sous

type (
	// Clusters is a collection of clusters.
	Clusters []Cluster
	// Cluster represents a logical deployment cluster.
	Cluster struct {
		// Kind is the kind of cluster, e.g. "Singularity"
		Kind string
		// URL is the base URL for this cluster
		URL string
		// DefaultEnv is the default environment variables to set for all tasks
		// running in this cluster.
		DefaultEnv EnvVars
	}
)
