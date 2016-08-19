package sous

import (
	"fmt"
	"strings"

	"github.com/samsalisbury/semv"
)

// Deployments returns all deployments described by the state.
func (s *State) Deployments() (Deployments, error) {
	ds := NewDeployments()
	Log.Vomit.Printf("%+v", s)
	Log.Vomit.Printf("%+v", s.Manifests)
	Log.Vomit.Printf("%#v", s.Manifests.Snapshot())
	for _, m := range s.Manifests.Snapshot() {
		deployments, err := s.DeploymentsFromManifest(m)
		Log.Vomit.Printf("%+v", deployments)
		if err != nil {
			return ds, err
		}
		conflict, ok := ds.AddAll(deployments)
		Log.Vomit.Printf("%+v", conflict)
		if !ok {
			return ds, fmt.Errorf("conflicting deploys: %s", conflict)
		}
	}
	for _, id := range ds.Keys() {
		d, _ := ds.Get(id)
		for name, val := range d.Cluster.Env {
			if _, ok := d.Env[name]; ok {
				continue
			}
			d.Env[name] = string(val)
		}
	}
	Log.Vomit.Printf("%+v", ds)
	return ds, nil
}

// Manifests creates manifests from deployments.
func (ds Deployments) Manifests() (Manifests, error) {
	ms := NewManifests()
	for _, d := range ds.Snapshot() {
		sl := d.SourceID.Location()
		m, ok := ms.Get(sl)
		if !ok {
			m = &Manifest{Deployments: DeploySpecs{}}
			for o := range d.Owners {
				// TODO: de-dupe or use a set on manifests.
				m.Owners = append(m.Owners, o)
			}
			m.Source = sl
		}
		spec := DeploySpec{
			Version:      d.SourceID.Version,
			DeployConfig: d.DeployConfig,
		}
		for k, v := range spec.DeployConfig.Env {
			clusterVal, ok := d.Cluster.Env[k]
			if !ok {
				continue
			}
			if string(clusterVal) == v {
				delete(spec.DeployConfig.Env, k)
			}
		}

		m.Deployments[d.Cluster.Name] = spec
		ms.Set(sl, m)
	}
	return ms, nil
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
	Log.Vomit.Printf("%+v", m)
	Log.Vomit.Println(m.Deployments)
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
		d, err := BuildDeployment(s, m, clusterName, spec, inherit)
		Log.Vomit.Println(d)
		if err != nil {
			return ds, err
		}
		ds.Add(d)
	}
	return ds, nil
}

// BuildDeployment constructs a deployment out of a Manifest.
func BuildDeployment(s *State, m *Manifest, nick string, spec DeploySpec, inherit []DeploySpec) (*Deployment, error) {
	ownMap := OwnerSet{}
	for i := range m.Owners {
		ownMap.Add(m.Owners[i])
	}
	ds := flattenDeploySpecs(append([]DeploySpec{spec}, inherit...))
	return &Deployment{
		ClusterName:  nick,
		Cluster:      s.Defs.Clusters[nick],
		DeployConfig: ds.DeployConfig,
		Owners:       ownMap,
		Kind:         m.Kind,
		SourceID:     m.Source.SourceID(ds.Version),
	}, nil
}

func flattenDeploySpecs(dss []DeploySpec) DeploySpec {
	var dcs []DeployConfig
	for _, s := range dss {
		dcs = append(dcs, s.DeployConfig)
	}
	ds := DeploySpec{DeployConfig: flattenDeployConfigs(dcs)}
	var zeroVersion semv.Version
	for _, s := range dss {
		if s.Version != zeroVersion {
			ds.Version = s.Version
			break
		}
	}
	for _, s := range dss {
		if s.clusterName != "" {
			ds.clusterName = s.clusterName
			break
		}
	}
	return ds
}
