package graph

import (
	sous "github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

// RefinedResolveFilter is a sous.ResolveFilter refined by user-requested flags.
type RefinedResolveFilter sous.ResolveFilter

func newRefinedResolveFilter(f *sous.ResolveFilter, discovered *SourceContextDiscovery) (*RefinedResolveFilter, error) {
	c := discovered.GetContext()
	if f == nil { // XXX I think this needs to be supplied anyway by consumers..
		f = &sous.ResolveFilter{}
	}
	repo := c.PrimaryRemoteURL
	offset := sous.ResolveFieldMatcher{Match: c.OffsetDir}

	if f.Repo != "" {
		repo = f.Repo
		offset = sous.ResolveFieldMatcher{All: true}
	}
	if repo == "" {
		return nil, errors.Errorf("no repo specified, please use -repo or run sous inside a git repo with a configured remote")
	}
	if !f.Offset.All && offset.Match == "" {
		offset = f.Offset
	}
	rrf := RefinedResolveFilter(*f)
	rrf.Repo = repo
	rrf.Offset = offset
	if f.Tag == "" {
		rrf.Tag = discovered.TagVersion()
	}
	return &rrf, nil
}

func newTargetManifestID(rrf *RefinedResolveFilter) (TargetManifestID, error) {
	if rrf == nil {
		return TargetManifestID{}, errors.Errorf("nil ResolveFilter")
	}
	if rrf.Repo == "" {
		return TargetManifestID{}, errors.Errorf("empty Repo")
	}
	return TargetManifestID{
		Source: sous.SourceLocation{
			Repo: rrf.Repo,
			Dir:  rrf.Offset.Match,
		},
		Flavor: rrf.Flavor.Match,
	}, nil
}

func newTargetManifest(auto userSelectedOTPLDeployManifest, tmid TargetManifestID, s *sous.State) TargetManifest {
	mid := sous.ManifestID(tmid)
	m, ok := s.Manifests.Get(mid)

	if ok {
		return TargetManifest{m}
	}

	var deploySpecs sous.DeploySpecs
	if auto.Manifest != nil {
		deploySpecs = auto.Manifest.Deployments
		m = auto.Clone()
	}
	if m == nil {
		m = &sous.Manifest{}
	}
	if len(deploySpecs) == 0 {
		deploySpecs = defaultDeploySpecs(s.Defs.Clusters)
	}

	m.Deployments = deploySpecs
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
