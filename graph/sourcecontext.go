package graph

import (
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func newTargetSourceLocation(f *config.DeployFilterFlags, c *sous.SourceContext) (TargetSourceLocation, error) {
	if c == nil {
		c = &sous.SourceContext{}
	}
	if f == nil {
		f = &config.DeployFilterFlags{}
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
			return TargetSourceLocation{}, errors.Errorf("you specified -offset but not -repo")
		}
		offset = f.Offset
	}
	if repo == "" {
		return TargetSourceLocation{}, errors.Errorf("no repo specified, please use -repo or run sous inside a git repo")
	}
	return TargetSourceLocation{
		Repo: repo,
		Dir:  offset,
	}, nil
}
