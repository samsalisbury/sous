package sous

import (
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/opentable/sous/util/logging"
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
			reportResourceMessage(fmt.Sprintf("Could not parse value: '%s' for cpus as a float, using default: %f", cpuStr, cpus), r, logging.Log)
		} else {
			reportDebugResourceMessage(fmt.Sprintf("Using default value for cpus: %f.", cpus), r, logging.Log)
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
			reportResourceMessage(fmt.Sprintf("Could not parse value: '%s' for memory as an int, using default: %f", memStr, memory), r, logging.Log)
		} else {
			reportDebugResourceMessage(fmt.Sprintf("Using default value for memory: %f.", memory), r, logging.Log)
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
			reportResourceMessage(fmt.Sprintf("Could not parse value: '%s' for ports as a int, using default: %d", portStr, ports), r, logging.Log)
		} else {
			reportDebugResourceMessage(fmt.Sprintf("Using default value for ports: %d", ports), r, logging.Log)
		}
	}
	return int32(ports)
}

// Equal checks equivalence between resource maps
func (r Resources) Equal(o Resources) bool {
	reportDebugResourceMessage(fmt.Sprintf("Comparing resources: %+ v ?= %+ v", r, o), r, logging.Log)
	if len(r) != len(o) {
		reportDebugResourceMessage("Lengths differ", r, logging.Log)
		return false
	}

	if r.Ports() != o.Ports() {
		reportDebugResourceMessage("Ports differ", r, logging.Log)
		return false
	}

	if math.Abs(r.Cpus()-o.Cpus()) > 0.001 {
		reportDebugResourceMessage("Cpus differ", r, logging.Log)
		return false
	}

	if math.Abs(r.Memory()-o.Memory()) > 0.001 {
		reportDebugResourceMessage("Memory differ", r, logging.Log)
		return false
	}

	return true
}

type resourceMessage struct {
	logging.CallerInfo
	msg        string
	ports      int32
	cpus       float64
	memory     float64
	isDebugMsg bool
}

func reportDebugResourceMessage(msg string, r Resources, log logging.LogSink) {
	reportResourceMessage(msg, r, log, true)
}

func reportResourceMessage(msg string, r Resources, log logging.LogSink, debug ...bool) {
	debugStmt := false
	if len(debug) > 0 {
		debugStmt = debug[0]
	}

	//not going to call Ports/Cpus/Memory to get values since those functions actually call reportResourceMessage
	var ports int32
	if portStr, present := r["ports"]; present {
		ports64, _ := strconv.ParseInt(portStr, 10, 32)
		ports = int32(ports64)
	}
	var memory float64
	if memStr, present := r["memory"]; present {
		memory, _ = strconv.ParseFloat(memStr, 64)
	}
	var cpus float64
	if cpuStr, present := r["cpus"]; present {
		cpus, _ = strconv.ParseFloat(cpuStr, 64)
	}

	msgLog := resourceMessage{
		msg:        msg,
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		ports:      ports,
		cpus:       cpus,
		memory:     memory,
		isDebugMsg: debugStmt,
	}
	logging.Deliver(msgLog, log)
}

func (msg resourceMessage) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "%s\n", msg.composeMsg())
}

func (msg resourceMessage) DefaultLevel() logging.Level {
	level := logging.WarningLevel
	if msg.isDebugMsg {
		level = logging.DebugLevel
	}

	return level
}

func (msg resourceMessage) Message() string {
	return msg.composeMsg()
}

func (msg resourceMessage) composeMsg() string {
	return fmt.Sprintf("%s: ports %v, cpus %v memory %v", msg.msg, msg.ports, msg.cpus, msg.memory)
}

func (msg resourceMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-generic-v1")
	f("ports", msg.ports)
	f("cpus", msg.cpus)
	f("memory", msg.memory)
	msg.CallerInfo.EachField(f)
}
