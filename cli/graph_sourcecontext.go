package cli

import (
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func resolveSourceLocation(f *DeployFilterFlags, c *sous.SourceContext) (sous.SourceLocation, error) {
	if c == nil {
		c = &sous.SourceContext{}
	}
	if f == nil {
		f = &DeployFilterFlags{}
	}
	var repo, offset = c.PrimaryRemoteURL, c.OffsetDir
	if f.Repo != "" {
		repo = f.Repo
	}
	if f.Repo != "" {
		repo = f.Repo
		offset = ""
	}
	if f.Offset != "" {
		if f.Repo == "" {
			return sous.SourceLocation{}, errors.Errorf("you specified -offset but not -repo")
		}
		offset = f.Offset
	}
	if repo == "" {
		return sous.SourceLocation{}, errors.Errorf("no repo specified, please use -repo or run sous inside a git repo")
	}
	return sous.SourceLocation{
		RepoURL:    repo,
		RepoOffset: offset,
	}, nil
}
