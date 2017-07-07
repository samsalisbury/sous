package sous

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

// Deployments returns all deployments described by the state.
func (s *State) Deployments() (Deployments, error) {
	return s.Manifests.Deployments(s.Defs)
}

// Deployments returns all deployments described by these Manifests.
func (ms Manifests) Deployments(defs Defs) (Deployments, error) {
	ds := NewDeployments()
	for _, m := range ms.Snapshot() {
		deployments, err := DeploymentsFromManifest(defs, m)
		if err != nil {
			return ds, err
		}
		conflict, ok := ds.AddAll(deployments)
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
		cluster, ok := defs.Clusters[d.ClusterName]
		if !ok {
			return ds, errors.Errorf("cluster %q is not described in defs.yaml (but specified in manifest %q)",
				d.ClusterName, id.ManifestID)
		}
		if cluster == nil {
			return ds, errors.Errorf("cluster %q is nil, check defs.yaml", d.ClusterName)
		}
		d.Cluster = cluster
	}
	return ds, nil
}

// PutbackManifests creates manifests from deployments.
func (ds Deployments) PutbackManifests(defs Defs, olds Manifests) (Manifests, error) {
	ms := NewManifests()
	for _, k := range ds.Keys() {
		d, _ := ds.Get(k)
		if d.ClusterName == "" {
			return ms, errors.Errorf("invalid deploy ID (no cluster name)")
		}
		if d.Cluster == nil {
			cluster, ok := defs.Clusters[d.ClusterName]
			if !ok {
				return ms, errors.Errorf("cluster %q is not described in defs.yaml", d.ClusterName)
			}
			d.Cluster = cluster
		}
		// Lookup the current manifest for this deployment.
		mid := d.ManifestID()

		m, ok := ms.Get(mid)
		old, was := olds.Get(mid)

		if !ok {
			m = &Manifest{Deployments: DeploySpecs{}}
			m.Owners = d.Owners.Slice()
			m.SetID(mid)
		}
		spec := DeploySpec{
			Version:      d.SourceID.Version,
			DeployConfig: d.DeployConfig.Clone(),
		}

		if was {
			if oldD, had := old.Deployments[d.ClusterName]; had {
				spec.DeployConfig.Startup = d.Cluster.Startup.UnmergeDefaults(spec.DeployConfig.Startup, oldD.Startup)
			}
		}

		for k, v := range spec.DeployConfig.Env {
			clusterVal, ok := d.Cluster.Env[k]
			if !ok {
				continue
			}
			if string(clusterVal) == v {
				Log.Debug.Printf("Redundant environment definition: %s=%s", k, v)
				if was {
					if oldSpec, had := old.Deployments[d.ClusterName]; had {
						if _, present := oldSpec.Env[k]; present {
							Log.Debug.Printf("Env pair %s=%s present in existing manifest: retained.", k, v)
						} else {
							Log.Debug.Printf("Env pair %s=%s absent in existing manifest: elided.", k, v)
							delete(spec.Env, k)
						}
					} else {
						Log.Debug.Printf("Cluster %q absent in existing manifest: eliding %s=%s.", d.ClusterName, k, v)
						delete(spec.Env, k)
					}
				} else {
					Log.Debug.Printf("Manifest for %v absent in existing manifest list: eliding %s=%s.", mid, k, v)
					delete(spec.Env, k)
				}
			}
		}
		m.Deployments[d.ClusterName] = spec
		m.Kind = d.Kind

		ms.Set(mid, m)
	}

	for _, k := range ms.Keys() {
		m, there := ms.Get(k)
		if !there {
			continue
		}
		ms.Set(k, m)
	}

	return ms, nil
}

// RawManifests creates manifests from deployments.
// "raw" because it's brand new - it doesn't maintain certain essential state over time.
// For almost all uses, you want PutbackManifests
func (ds Deployments) RawManifests(defs Defs) (Manifests, error) {
	ms := NewManifests()
	for _, k := range ds.Keys() {
		d, _ := ds.Get(k)
		if d.ClusterName == "" {
			return ms, errors.Errorf("invalid deploy ID (no cluster name)")
		}
		if d.Cluster == nil {
			cluster, ok := defs.Clusters[d.ClusterName]
			if !ok {
				return ms, errors.Errorf("cluster %q is not described in defs.yaml", d.ClusterName)
			}
			d.Cluster = cluster
		}
		// Lookup the current manifest for this deployment.
		mid := d.ManifestID()
		m, ok := ms.Get(mid)
		if !ok {
			m = &Manifest{Deployments: DeploySpecs{}}
			m.Owners = d.Owners.Slice()
			m.SetID(mid)
		}
		spec := DeploySpec{
			Version:      d.SourceID.Version,
			DeployConfig: d.DeployConfig.Clone(),
		}
		for k, v := range spec.DeployConfig.Env {
			clusterVal, ok := d.Cluster.Env[k]
			if !ok {
				continue
			}
			if string(clusterVal) == v {
				Log.Debug.Printf("Redundant environment definition: %s=%s", k, v)
			}
		}
		m.Deployments[d.ClusterName] = spec
		m.Kind = d.Kind

		ms.Set(mid, m)
	}

	for _, k := range ms.Keys() {
		m, there := ms.Get(k)
		if !there {
			continue
		}
		ms.Set(k, m)
	}

	return ms, nil
}

// DeploymentsFromManifest returns all deployments described by a single
// manifest, in terms of the wider state (i.e. global and cluster definitions
// and configuration).
func DeploymentsFromManifest(defs Defs, m *Manifest) (Deployments, error) {
	ds := NewDeployments()
	var inherit []DeploySpec

	for clusterName, spec := range m.Deployments {
		cluster, ok := defs.Clusters[clusterName]
		if !ok {
			return ds, errors.Errorf("cluster %q not described in defs.yaml", clusterName)
		}
		spec.clusterName = cluster.BaseURL
		d, err := BuildDeployment(defs, m, clusterName, spec, inherit)
		if err != nil {
			return ds, err
		}
		ds.Add(d)
	}
	return ds, nil
}

// BuildDeployment constructs a deployment out of a Manifest.
func BuildDeployment(defs Defs, m *Manifest, nick string, spec DeploySpec, inherit []DeploySpec) (*Deployment, error) {
	ownMap := NewOwnerSet(m.Owners...)
	cluster := defs.Clusters[nick]

	ds := flattenDeploySpecs(append([]DeploySpec{spec}, inherit...))
	ds.Startup = cluster.Startup.MergeDefaults(ds.Startup)

	// XXX Env merging belongs here

	return &Deployment{
		ClusterName:  nick,
		Cluster:      cluster,
		DeployConfig: ds.DeployConfig,
		Flavor:       m.Flavor,
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
	/*
		DSs have to be unique by clusterName, nicht war?
			for _, s := range dss {
				if s.clusterName != "" {
					ds.clusterName = s.clusterName
					break
				}
			}
	*/
	return ds
}
