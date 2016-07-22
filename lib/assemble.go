package sous

import (
	"fmt"
	"strings"
)

// Deployments returns all deployments described by the state.
func (s *State) Deployments() (Deployments, error) {
	ds := Deployments{}
	for _, m := range s.Manifests {
		deployments, err := s.DeploymentsFromManifest(m)
		if err != nil {
			return nil, err
		}
		ds = append(ds, deployments...)
	}
	return ds, nil
}

// DeploymentsFromManifest returns all deployments described by a single
// manifest, in terms of the wider state (i.e. global and cluster definitions
// and configuration).
func (s *State) DeploymentsFromManifest(m *Manifest) ([]*Deployment, error) {
	ds := []*Deployment{}
	var inherit []PartialDeploySpec
	if global, ok := m.Deployments["Global"]; ok {
		inherit = append(inherit, global)
	}
	for clusterName, spec := range m.Deployments {
		n, ok := s.Defs.Clusters[clusterName]
		if !ok {
			us := make([]string, 0, len(s.Defs.Clusters))
			for n := range s.Defs.Clusters {
				us = append(us, n)
			}
			return nil, fmt.Errorf("Could not find an cluster configured for name '%s' in [%s] (for %+v)", clusterName, strings.Join(us, ", "), m)
		}
		spec.clusterName = n.BaseURL
		d, err := BuildDeployment(m, clusterName, spec, inherit)
		if err != nil {
			return nil, err
		}
		ds = append(ds, d)
	}
	return ds, nil
}
