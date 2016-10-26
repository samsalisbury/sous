package sous

import (
	"math"
	"strconv"
)

// Resources is a mapping of resource name to value, used to provision
// single instances of an application. It is validated against
// State.Defs.Resources. The keys must match defined resource names, and the
// values must parse to the defined types.
type Resources map[string]string

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
