package cli

import (
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func newTargetSourceLocation(f *DeployFilterFlags, c *sous.SourceContext) (TargetManifestID, error) {
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
			return TargetManifestID{}, errors.Errorf("you specified -offset but not -repo")
		}
		offset = f.Offset
	}
	if repo == "" {
		return TargetManifestID{}, errors.Errorf("no repo specified, please use -repo or run sous inside a git repo")
	}
	return TargetManifestID{
		Source: sous.SourceLocation{
			Repo: repo,
			Dir:  offset,
		},
		Flavor: f.Flavor,
	}, nil
}
