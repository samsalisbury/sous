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
	offset := sous.ResolveFieldMatcher{}
	if c.OffsetDir != "" {
		offset = sous.NewResolveFieldMatcher(c.OffsetDir)
	}

	if f.Repo != "" {
		repo = f.Repo
		offset = sous.ResolveFieldMatcher{}
	}
	if repo == "" {
		return nil, errors.Errorf("no repo specified, please use -repo or run sous inside a git repo with a configured remote")
	}
	if !f.Offset.All() && offset.All() {
		offset = f.Offset
	}
	rrf := &(*f)
	rrf.Repo = repo
	rrf.Offset = offset
	if f.Tag.All() && discovered.TagVersion() != "" {
		rrf.SetTag(discovered.TagVersion())
	}
	return (*RefinedResolveFilter)(rrf), nil
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
			Dir:  rrf.Offset.ValueOr(""),
		},
		Flavor: rrf.Flavor.ValueOr(""),
	}, nil
}

// QueryMap returns a map suitable for use with the HTTP API.
func (mid TargetManifestID) QueryMap() map[string]string {
	manifestQuery := map[string]string{}
	manifestQuery["repo"] = mid.Source.Repo
	manifestQuery["offset"] = mid.Source.Dir
	manifestQuery["flavor"] = mid.Flavor
	return manifestQuery
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
		spec := sous.DeploySpec{
			DeployConfig: sous.DeployConfig{
				Resources: sous.Resources{},
				Env:       map[string]string{},
				Startup: sous.Startup{
					// should be defaults in the clusters, but we want to make these clear and explicit
					CheckReadyProtocol: "HTTP",
					CheckReadyURIPath:  "/health",
				},
				// XXX Should be 0 - used when no config has been specified
				NumInstances: 1,
			},
		}

		// repairing the validation flaws on a fresh DeploySpec sets defaults.
		// more importantly, this is a single consistent way to set those defaults.
		sous.RepairAll(spec.Validate())
		defaults[name] = spec
	}
	return defaults
}
