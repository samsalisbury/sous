//+build smoke

package smoke

import sous "github.com/opentable/sous/lib"

func setMemAndCPUForAll(ds sous.DeploySpecs) sous.DeploySpecs {
	for c := range ds {
		ds[c].Resources["memory"] = "1"
		ds[c].Resources["cpus"] = "0.001"
	}
	return ds
}

func setMinimalMemAndCPUNumInst1(m sous.Manifest) sous.Manifest {
	return transformEachDeployment(m, func(d sous.DeploySpec) sous.DeploySpec {
		d.Resources["memory"] = "1"
		d.Resources["cpus"] = "0.001"
		d.NumInstances = 1
		return d
	})
}

func setMinimalMemAndCPUNumInst0(m sous.Manifest) sous.Manifest {
	return transformEachDeployment(m, func(d sous.DeploySpec) sous.DeploySpec {
		d.Resources["memory"] = "1"
		d.Resources["cpus"] = "0.001"
		d.NumInstances = 0
		return d
	})
}

func transformEachDeployment(m sous.Manifest, f func(sous.DeploySpec) sous.DeploySpec) sous.Manifest {
	for c, d := range m.Deployments {
		m.Deployments[c] = f(d)
	}
	return m
}
