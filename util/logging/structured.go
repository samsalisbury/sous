package logging

import "github.com/sirupsen/logrus"

// LogMessage records a message to one or more structured logs
func (ls *LogSet) LogMessage(lvl Level, msg LogMessage) {
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

func (ls *LogSet) eachField(f FieldReportFn) {
	f("component-id", ls.name)
}
