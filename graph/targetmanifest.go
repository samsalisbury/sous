package graph

import (
	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func newTargetManifestID(f *config.DeployFilterFlags, discovered *SourceContextDiscovery) (TargetManifestID, error) {
	c := discovered.GetContext()
	if f == nil { // XXX I think this needs to be supplied anyway by consumers...
		f = &config.DeployFilterFlags{}
	}
	var repo, offset = c.PrimaryRemoteURL, c.OffsetDir
	if f.Repo != "" {
		repo = f.Repo
		offset = ""
	}
	if f.Offset != "" {
		if f.Repo == "" {
			return TargetManifestID{}, errors.Errorf("-offset doesn't make sense without a -repo or workspace remote")
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
		deploySpecs = defaultDeploySpecs(s.Defs.Clusters)
	}

	m = &sous.Manifest{
		Deployments: deploySpecs,
	}
	m.SetID(mid)

	fls := m.Validate()
	sous.RepairAll(fls)
	return TargetManifest{m}
}

func defaultDeploySpecs(clusters sous.Clusters) sous.DeploySpecs {
	defaults := sous.DeploySpecs{}
	for name := range clusters {
		defaults[name] = sous.DeploySpec{
			DeployConfig: sous.DeployConfig{
				Resources:    sous.Resources{},
				Env:          map[string]string{},
				NumInstances: 1,
			},
		}
	}
	return defaults
}
