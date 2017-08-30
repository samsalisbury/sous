package logging

import "github.com/sirupsen/logrus"

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
	}
}

func (ls LogSet) eachField(f FieldReportFn) {
	f("component-id", ls.name)
	/*
		 "@timestamp":
		    type: timestamp
		    description: Core timestamp field, used by Logstash and Elasticsearch for time indexing

		  "@uuid":
		    type: uuid
		    description: Message and Document ID, used for (idempotent) indexing in Elasticsearch

		  ot-env: string            # machine environment, prod-sc
		  ot-env-type: string       # environment type, prod
		  ot-env-location: string   # machine location, sc
		  host: string              # server host name
		  sequence-number: long     # counter for the messages sent by the logger since the app started

			// These are all optional:
		  service-type:             # mostly meant to match the discovery service name
		    type: string
		  instance-no:              # singularity instance number
		    type: int
		  session-id:
		    type: string
		  ot-env-flavor:
		    type: string
		  log-name:
		    type: string
		  logger-name:
		    type: string
		  logging-library-name:
		    type: string
		  logging-library-version:
		    type: string
		  thread-name:
		    type: string
		  application-version:      # distinguish the deployment that's logging; e.g. TC id, package.json...
		    type: string
		  singularity-task-id:
		    type: string
	*/
}
