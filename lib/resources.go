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
	cpuStr, present := r["cpus"]
	cpus, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		cpus = 0.1
		if present {
			Log.Warn.Printf("Could not parse value: '%s' for cpus as a float, using default: %f", cpuStr, cpus)
		} else {
			Log.Vomit.Printf("Using default value for cpus: %f.", cpus)
		}
	}
	return cpus
}

// Memory returns memory in MB.
func (r Resources) Memory() float64 {
	memStr, present := r["memory"]
	memory, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		memory = 100
		if present {
			Log.Warn.Printf("Could not parse value: '%s' for memory as an int, using default: %f", memStr, memory)
		} else {
			Log.Vomit.Printf("Using default value for memory: %f.", memory)
		}
	}
	return memory
}

// Ports returns the number of ports required.
func (r Resources) Ports() int32 {
	portStr, present := r["ports"]
	ports, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		ports = 1
		if present {
			Log.Warn.Printf("Could not parse value: '%s' for ports as a int, using default: %d", portStr, ports)
		} else {
			Log.Vomit.Printf("Using default value for ports: %d", ports)
		}
	}
	return int32(ports)
}

// Equal checks equivalence between resource maps
func (r Resources) Equal(o Resources) bool {
	Log.Vomit.Printf("Comparing resources: %+ v ?= %+ v", r, o)
	if len(r) != len(o) {
		Log.Vomit.Println("Lengths differ")
		return false
	}

	if r.Ports() != o.Ports() {
		Log.Vomit.Println("Ports differ")
		return false
	}

	if math.Abs(r.Cpus()-o.Cpus()) > 0.001 {
		Log.Vomit.Println("Cpus differ")
		return false
	}

	if math.Abs(r.Memory()-o.Memory()) > 0.001 {
		Log.Vomit.Println("Memory differ")
		return false
	}

	return true
}
