package core

import "fmt"

type CompiledDatacentre struct {
	Datacentre
	Manifests DatacentreManifests
}

type DatacentreManifests map[string]DatacentreManifest

type DatacentreManifest struct {
	Deployment
	App App
}

// DatacentreView returns a state orgenised by datacentre.
func (s *MergedState) CompiledDatacentre(name string) CompiledDatacentre {
	// Return a state object that excludes everything
	// not relevant to the named dc.
	dc, ok := s.Datacentres[name]
	if !ok {
		panic(fmt.Sprintf("Datacentre %q not defined", dc))
	}
	ms := DatacentreManifests{}
	for appName, m := range s.Manifests {
		if d, ok := m.Deployments[name]; ok {
			manifest := DatacentreManifest{d, m.App}
			ms[appName] = manifest
		}
	}
	return CompiledDatacentre{*dc, ms}
}
