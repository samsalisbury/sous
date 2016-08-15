package cli

import "github.com/opentable/sous/lib"

// DeployFilterFlags are CLI flags used to configure the underlying deployments
// a given command will refer to
// N.b. that not every command will use every filter
type DeployFilterFlags struct {
	Repo     string
	Offset   string
	Tag      string
	Revision string
	Cluster  string
	All      bool
}

func (f *DeployFilterFlags) buildPredicate() sous.DeploymentPredicate {
	var preds []sous.DeploymentPredicate

	if f.All {
		return func(*sous.Deployment) bool { return true }
	}

	if f.Repo != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.RepoURL == f.Repo
		})
	}

	if f.Offset != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.RepoOffset == f.Offset
		})
	}

	if f.Tag != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.Tag() == f.Tag
		})
	}

	if f.Revision != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.RevID() == f.Revision
		})
	}

	if f.Cluster != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.ClusterNickname == f.Cluster
		})
	}

	switch len(preds) {
	case 0:
		return nil
	case 1:
		return preds[0]
	default:
		return func(d *sous.Deployment) bool {
			for _, f := range preds {
				if !f(d) { // AND(preds...)
					return false
				}
			}
			return true
		}
	}
}

const (
	repoFlagHelp = `
	-repo REPOSITORY_NAME
		source code repository location

		The repository context is the name of a source code repository whose
		code, configuration, artifacts, deployments, etc. will be acted upon.
		If sous is run from inside a Git repository, then repo will default to
		the normalised git-configured fetch URL of any remote named "upstream"
		or "origin", in that order.

		Sous uses go-style repository URLs, and currently only supports GitHub-
		based repositories, e.g. "github.com/user/repo"

		`

	offsetFlagHelp = `
	-offset RELATIVE_PATH
		source code relative repository offset

		Repository context offset is the relative path within a repository where
		a piece of software is defined.

		If you are working in a subdirectory of a repository, the default value
		for offset will be the relative path of the current working directory
		from the repository root.

		Note: if you supply the -repo flag but not -offset, then -offset
		defaults to "".

		`
	tagFlagHelp = `
	-tag TAG_NAME
		source code revision tag

		Repository tag is the name of a tag in the repository to act upon.

		`
	revisionFlagHelp = `
	-revision REVISION_ID
		source code revision ID

		Revision ID is the ID of a revision in the repository to act upon.

		`

	clusterFlagHelp = `
	-cluster CLUSTER
	  target deployment cluster

		Cluster name is the the deployment environment to consider

		`

	allFlagHelp = `
	-all
	  all deployments should be considered

	  `
)

var (
	sourceFlagsHelp        = repoFlagHelp + offsetFlagHelp + tagFlagHelp + revisionFlagHelp
	rectifyFilterFlagsHelp = repoFlagHelp + offsetFlagHelp + clusterFlagHelp + allFlagHelp
)
