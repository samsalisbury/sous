package logging

import (
	"os"
	"strconv"
	"sync/atomic"

	"github.com/samsalisbury/semv"
)

// applicationID tries to identify this instance of the application.
// it captures a number of details that are specific to Sous.
type applicationID struct {
	otenv,
	otenvtype,
	otenvlocation string

	host               string
	singularitytaskid  string
	applicationversion string
	instanceno         uint
	sequenceNumber     uint64
}

func collectAppID(version semv.Version, env map[string]string) *applicationID {
	id := applicationID{}
	getenv := func(n string) string {
		v, found := env[n]
		if !found {
			return "unknown"
		}
		return v
	}
	id.otenv = getenv("OT_ENV")
	id.otenvtype = getenv("OT_ENV_TYPE")
	id.otenvlocation = getenv("OT_ENV_LOCATION")
	id.singularitytaskid = getenv("TASK_ID")
	if i, err := strconv.ParseUint(getenv("INSTANCE_NO"), 10, 32); err == nil {
		id.instanceno = uint(i)
	}
	host, err := os.Hostname()
	id.host = host
	if err != nil {
		id.host = "unknown: " + err.Error()
	}

	id.applicationversion = version.String()

	return &id
}

// seq returns the next sequence number in a concurrency-safe manner.
func (id *applicationID) seq() uint64 {
	return atomic.AddUint64(&id.sequenceNumber, 1)
}

func (id *applicationID) EachField(f FieldReportFn) {
	f(SequenceNumber, id.seq())
	f(OtEnv, id.otenv)
	f(OtEnvType, id.otenvtype)
	f(OtEnvLocation, id.otenvlocation)
	f(Host, id.host)
	f(InstanceNo, id.instanceno)
	f(ApplicationVersion, id.applicationversion)
	f(SingularityTaskId, id.singularitytaskid)
	f(ServiceType, "sous")
}

func (id *applicationID) metricsScope() string {
	return id.otenvtype + "." + id.otenvlocation
}
