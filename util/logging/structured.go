package logging

import (
	"github.com/pborman/uuid"
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

	logto = logto.WithField("severity", lvl.String())

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
	case ExtraDebug1Level:
		logto.Debug(msg.Message())
	}
}
func (ls LogSet) eachField(f FieldReportFn) {
	if ls.appRole != "" {
		f("component-id", "sous-"+ls.appRole)
	} else {
		f("component-id", "sous")
	}
	f("logger", ls.name)
	f("@uuid", uuid.New())

	ls.appIdent.EachField(f)

	/*
	 "@timestamp":
	    type: timestamp
	    description: Core timestamp field, used by Logstash and Elasticsearch for time indexing

	*/
}
