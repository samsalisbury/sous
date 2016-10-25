package sous

import "fmt"

type (
	// DeployConfig represents the configuration of a deployment's tasks,
	// in a specific cluster. i.e. their resources, environment, and the number
	// of instances.
	DeployConfig struct {
		// Resources represents the resources each instance of this software
		// will be given by the execution environment.
		Resources Resources `yaml:",omitempty" validate:"keys=nonempty,values=nonempty"`
		// Env is a list of environment variables to set for each instance of
		// of this deployment. It will be checked for conflict with the
		// definitions found in State.Defs.EnvVars, and if not in conflict
		// assumes the greatest priority.
		Args []string `yaml:",omitempty" validate:"values=nonempty"`
		Env  `yaml:",omitempty" validate:"keys=nonempty,values=nonempty"`
		// NumInstances is a guide to the number of instances that should be
		// deployed in this cluster, note that the actual number may differ due
		// to decisions made by Sous. If set to zero, Sous will decide how many
		// instances to launch.
		NumInstances int

		// Volumes lists the volume mappings for this deploy
		Volumes Volumes
	}

	// Env is a mapping of environment variable name to value, used to provision
	// single instances of an application.
	Env map[string]string
)

func (dc *DeployConfig) String() string {
	return fmt.Sprintf("#%d %+v : %+v %+v", dc.NumInstances, dc.Resources, dc.Env, dc.Volumes)
}

// Equal is used to compare DeployConfigs
func (dc *DeployConfig) Equal(o DeployConfig) bool {
	Log.Vomit.Printf("%+ v ?= %+ v", dc, o)
	return (dc.NumInstances == o.NumInstances &&
		dc.Env.Equal(o.Env) &&
		dc.Resources.Equal(o.Resources) &&
		dc.Volumes.Equal(o.Volumes))
}

// Diff returns a list of differences between this and the other DeployConfig.
func (dc *DeployConfig) Diff(o DeployConfig) (bool, []string) {
	var diffs []string
	if dc.NumInstances != o.NumInstances {
		diffs = append(diffs, fmt.Sprintf("number of instances; this: %d; other: %d", dc.NumInstances, o.NumInstances))
	}
	if !dc.Env.Equal(o.Env) {
		diffs = append(diffs, fmt.Sprintf("env; this: %v; other: %v", dc.Env, o.Env))
	}
	if !dc.Resources.Equal(o.Resources) {
		diffs = append(diffs, fmt.Sprintf("resources; this: %v; other: %v", dc.Resources, o.Resources))
	}
	if !dc.Volumes.Equal(o.Volumes) {
		diffs = append(diffs, fmt.Sprintf("volumes; this: %v; other: %v", dc.Volumes, o.Volumes))
	}
	return len(diffs) == 0, diffs
}

func (dc DeployConfig) Clone() (c DeployConfig) {
	c.NumInstances = dc.NumInstances
	c.Args = make([]string, len(dc.Args))
	copy(dc.Args, c.Args)
	c.Env = make(Env)
	for k, v := range dc.Env {
		c.Env[k] = v
	}
	c.Resources = make(Resources)
	for k, v := range dc.Resources {
		c.Resources[k] = v
	}
	c.Volumes = make(Volumes, len(dc.Volumes))
	copy(dc.Volumes, c.Volumes)
	return
}

// Equal compares Envs
func (e Env) Equal(o Env) bool {
	Log.Vomit.Printf("Envs: %+ v ?= %+ v", e, o)
	if len(e) != len(o) {
		Log.Vomit.Printf("Envs: %+ v != %+ v (%d != %d)", e, o, len(e), len(o))
		return false
	}

	for name, value := range e {
		if ov, ok := o[name]; !ok || ov != value {
			Log.Vomit.Printf("Envs: %+ v != %+ v [%q] %q != %q", e, o, name, value, ov)
			return false
		}
	}
	Log.Vomit.Printf("Envs: %+ v == %+ v !", e, o)
	return true
}

func flattenDeployConfigs(dcs []DeployConfig) DeployConfig {
	dc := DeployConfig{
		Resources: make(Resources),
		Env:       make(Env),
	}
	for _, c := range dcs {
		if c.NumInstances != 0 {
			dc.NumInstances = c.NumInstances
			break
		}
	}
	for _, c := range dcs {
		if len(c.Volumes) != 0 {
			dc.Volumes = c.Volumes
			break
		}
	}
	for _, c := range dcs {
		if len(c.Args) != 0 {
			dc.Args = c.Args
			break
		}
	}
	for _, c := range dcs {
		for n, v := range c.Resources {
			if _, set := dc.Resources[n]; !set {
				dc.Resources[n] = v
			}
		}
		for n, v := range c.Env {
			if _, set := dc.Env[n]; !set {
				dc.Env[n] = v
			}
		}
	}
	return dc
}
