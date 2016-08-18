package sous

import (
	"fmt"
	"log"
	"strings"
)

// Deployments returns all deployments described by the state.
func (s *State) Deployments() (Deployments, error) {
	ds := NewDeployments()
	for _, m := range s.Manifests.Snapshot() {
		deployments, err := s.DeploymentsFromManifest(m)
		if err != nil {
			return ds, err
		}
		conflict, ok := ds.AddAll(deployments)
		if !ok {
			return ds, fmt.Errorf("conflicting deploys: %s", conflict)
		}
	}
	log.Println("OK", ds.Keys())
	return ds, nil
}

// DeploymentsFromManifest returns all deployments described by a single
// manifest, in terms of the wider state (i.e. global and cluster definitions
// and configuration).
func (s *State) DeploymentsFromManifest(m *Manifest) (Deployments, error) {
	ds := NewDeployments()
	var inherit []DeploySpec
	if global, ok := m.Deployments["Global"]; ok {
		inherit = append(inherit, global)
		delete(m.Deployments, "Global")
	}
	for clusterName, spec := range m.Deployments {
		n, ok := s.Defs.Clusters[clusterName]
		if !ok {
			us := make([]string, 0, len(s.Defs.Clusters))
			for n := range s.Defs.Clusters {
				us = append(us, n)
			}
			return ds, fmt.Errorf("no cluster %q in [%s] (for %+v)",
				clusterName, strings.Join(us, ", "), m)
		}
		spec.clusterName = n.BaseURL
		d, err := BuildDeployment(m, clusterName, spec, inherit)
		if err != nil {
			return ds, err
		}
		ds.Add(d)
	}
	return ds, nil
}
