package sous

import (
	"fmt"
	"math"
	"strconv"
)

type (
	// Resources is a mapping of resource name to value, used to provision
	// single instances of an application. It is validated against
	// State.Defs.Resources. The keys must match defined resource names, and the
	// values must parse to the defined types.
	Resources map[string]string

	// A MissingResourceFlaw captures the absence of a required resource field,
	// and tries to repair it from the state defaults
	MissingResourceFlaw struct {
		Resources
		did            *DeploymentID
		ClusterName    string
		Field, Default string
	}
)

// Clone returns a deep copy of this Resources.
func (r Resources) Clone() Resources {
	rs := make(Resources, len(r))
	for name, value := range r {
		rs[name] = value
	}
	return rs
}

// AddContext implements Flaw.AddContext.
func (f *MissingResourceFlaw) AddContext(name string, i interface{}) {
	if name == "cluster" {
		if name, is := i.(string); is {
			f.ClusterName = name
		}
	}
	if name == "deployment" {
		if dep, is := i.(*Deployment); is {
			did := dep.ID()
			f.did = &did
		}
	}
	/*
		// I'd misremembered that the State.Defs held the GDM-wide defaults
		// which isn't true. Leaving this here to sort of demostrate the idea
		if name != "state" {
			return
		}
		if state, is := i.(*State); is {
			f.State = state
		}
	*/
}

func (f *MissingResourceFlaw) String() string {
	if f.did != nil {
		return fmt.Sprintf("Missing resource field %q for deployment %s", f.Field, f.did)
	}
	name := f.ClusterName
	if name == "" {
		name = "??"
	}

	return fmt.Sprintf("Missing resource field %q for cluster %s", f.Field, name)
}

// Repair adds all missing fields set to default values.
func (f *MissingResourceFlaw) Repair() error {
	f.Resources[f.Field] = f.Default
	return nil
}

// Validate checks that each required resource value is set in this Resources,
// or in the inherited Resources.
func (r Resources) Validate() []Flaw {
	var flaws []Flaw

	if f := r.validateField("cpus", "0.1"); f != nil {
		flaws = append(flaws, f)
	}
	if f := r.validateField("memory", "100"); f != nil {
		flaws = append(flaws, f)
	}
	if f := r.validateField("ports", "1"); f != nil {
		flaws = append(flaws, f)
	}

	return flaws
}

func (r Resources) validateField(name, def string) Flaw {
	if _, has := r[name]; !has {
		return &MissingResourceFlaw{Resources: r, Field: name, Default: def}
	}
	return nil
}

// Cpus returns the number of CPUs.
func (r Resources) Cpus() float64 {
	cpuStr := r["cpus"]
	cpus, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		cpus = 0.1
	}
	return cpus
}

// Memory returns memory in MB.
func (r Resources) Memory() float64 {
	memStr := r["memory"]
	memory, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		memory = 100
	}
	return memory
}

// Ports returns the number of ports required.
func (r Resources) Ports() int32 {
	portStr := r["ports"]
	ports, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		ports = 1
	}
	return int32(ports)
}

// Equal checks equivalence between resource maps
func (r Resources) Equal(o Resources) bool {
	if len(r) != len(o) {
		return false
	}

	if r.Ports() != o.Ports() {
		return false
	}

	if math.Abs(r.Cpus()-o.Cpus()) > 0.001 {
		return false
	}

	if math.Abs(r.Memory()-o.Memory()) > 0.001 {
		return false
	}

	return true
}
