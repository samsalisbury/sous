package logging

import (
	"os"

	"github.com/pborman/uuid"
	"github.com/samsalisbury/semv"
	"github.com/sirupsen/logrus"
)

// LogMessage records a message to one or more structured logs
func (ls LogSet) LogMessage(lvl Level, msg LogMessage) {
	logto := logrus.FieldLogger(ls.logrus)

	ls.eachField(func(name string, value interface{}) {
		logto = logto.WithField(name, value)
	})

	msg.EachField(func(name string, value interface{}) {
		logto = logto.WithField(name, value)
	})

	switch lvl {
	default:
		logto.Printf("unknown Level: %d - %q", lvl, msg.Message())
	case CriticalLevel:
		logto.Error(msg.Message())
	case WarningLevel:
		logto.Warn(msg.Message())
	case InformationLevel:
		logto.Info(msg.Message())
	case DebugLevel:
		logto.Debug(msg.Message())
	case ExtraDebugLevel1:
		logto.Debug(msg.Message())
	}
}

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

func collectAppID(version semv.Version) *applicationID {
	id := applicationID{}
	getenv := func(n string) string {
		v, found := os.LookupEnv("OTENV")
		if !found {
			return "unknown"
		}
		return v
	}
	id.otenv = getenv("OT_ENV")
	id.otenvtype = getenv("OT_ENV_TYPE")
	id.otenvlocation = getenv("OT_ENV_LOCATION")
	id.singularitytaskid = getenv("TASK_ID")

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

func (ls LogSet) eachField(f FieldReportFn) {
	f("component-id", ls.name)
	f("@uuid", uuid.New())

	ls.appIdent.EachField(f)

	/*
	 "@timestamp":
	    type: timestamp
	    description: Core timestamp field, used by Logstash and Elasticsearch for time indexing

	*/
}
