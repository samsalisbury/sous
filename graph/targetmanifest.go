package graph

import (
	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func newTargetManifestID(f *config.DeployFilterFlags, g GitSourceContext) (TargetManifestID, error) {
	c, err := g.SourceContext()
	if err != nil {
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

func newTargetManifest(auto UserSelectedOTPLDeploySpecs, tmid TargetManifestID, s *sous.State) TargetManifest {
	mid := sous.ManifestID(tmid)
	m, ok := s.Manifests.Get(mid)
	if ok {
		return TargetManifest{m}
	}
	deploySpecs := auto.DeploySpecs
	if len(deploySpecs) == 0 {
		deploySpecs = defaultDeploySpecs()
	}

	m = &sous.Manifest{
		Deployments: deploySpecs,
	}
	m.SetID(mid)
	return TargetManifest{m}
}

func defaultDeploySpecs() sous.DeploySpecs {
	return sous.DeploySpecs{
		"Global": {
			DeployConfig: sous.DeployConfig{
				Resources:    sous.Resources{},
				Env:          map[string]string{},
				NumInstances: 3,
			},
		},
	}
}
