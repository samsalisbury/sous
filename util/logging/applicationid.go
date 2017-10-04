package logging

import (
	"os"
	"strconv"

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
	sequenceNumber     uint
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

func (id *applicationID) EachField(f FieldReportFn) {
	id.sequenceNumber++
	f("sequence-number", id.sequenceNumber)
	f("ot-env", id.otenv)
	f("ot-env-type", id.otenvtype)
	f("ot-env-location", id.otenvlocation)
	f("host", id.host)
	f("instance-no", id.instanceno)
	f("application-version", id.applicationversion)
	f("singularity-task-id", id.singularitytaskid)
	f("service-type", "sous")

	/*
		  ot-env: string            # machine environment, prod-sc
		  ot-env-type: string       # environment type, prod
		  ot-env-location: string   # machine location, sc
		  host: string              # server host name
		  sequence-number: long     # counter for the messages sent by the logger since the app started
		  instance-no:              # singularity instance number
			service-type:             # mostly meant to match the discovery service name
		  application-version:      # distinguish the deployment that's logging; e.g. TC id, package.json...
		  singularity-task-id:
	*/
}
