//+build smoke

package smoke

import sous "github.com/opentable/sous/lib"

type ManifestTransform func(sous.Manifest) sous.Manifest

func setMemAndCPUForAll(ds sous.DeploySpecs) sous.DeploySpecs {
	for c := range ds {
		d := ds[c]
		d.Resources["memory"] = "1"
		d.Resources["cpus"] = "0.001"
		d.Startup.ConnectDelay = 0
		ds[c] = d
	}
	return ds
}

func setMinimalMemAndCPUNumInst1(m sous.Manifest) sous.Manifest {
	return transformEachDeployment(m, func(d sous.DeploySpec) sous.DeploySpec {
		d.Resources["memory"] = "1"
		d.Resources["cpus"] = "0.001"
		d.NumInstances = 1
		d.Startup.ConnectDelay = 0
		return d
	})
}

func setMinimalMemAndCPUNumInst0(m sous.Manifest) sous.Manifest {
	return transformEachDeployment(m, func(d sous.DeploySpec) sous.DeploySpec {
		d.Resources["memory"] = "1"
		d.Resources["cpus"] = "0.001"
		d.NumInstances = 0
		d.Startup.ConnectDelay = 0
		return d
	})
}

func transformEachDeployment(m sous.Manifest, f func(sous.DeploySpec) sous.DeploySpec) sous.Manifest {
	for c, d := range m.Deployments {
		m.Deployments[c] = f(d)
	}
	return m
}
