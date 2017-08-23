package logging

import "github.com/Sirupsen/logrus"

// LogMessage records a message to one or more structured logs
func (ls *LogSet) LogMessage(lvl level, msg logMessage) {
	logto := logrus.FieldLogger(ls.logrus)

	msg.eachField(func(name string, value interface{}) {
		logto = logto.WithField(name, value)
	})

	switch lvl {
	default:
		logto.Printf("unknown level: %d - %q", msg.message())
	case criticalLevel:
		logto.Error(msg.message())
	case warningLevel:
		logto.Warn(msg.message())
	case informationLevel:
		logto.Info(msg.message())
	case debugLevel:
		logto.Debug(msg.message())
	}
}
