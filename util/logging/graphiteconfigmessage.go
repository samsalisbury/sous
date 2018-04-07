package logging

import (
	"time"

	graphite "github.com/cyberdelia/go-metrics-graphite"
)

type graphiteConfigMessage struct {
	CallerInfo
	cfg *graphite.Config
}

func reportGraphiteConfig(cfg *graphite.Config, ls LogSink) {
	msg := graphiteConfigMessage{
		CallerInfo: GetCallerInfo(NotHere()),
		cfg:        cfg,
	}
	Deliver(msg, ls)
}

func (gcm graphiteConfigMessage) DefaultLevel() Level {
	return InformationLevel
}

func (gcm graphiteConfigMessage) Message() string {
	if gcm.cfg == nil {
		return "Not connecting to Graphite server"
	}
	return "Connecting to Graphite server"
}

func (gcm graphiteConfigMessage) EachField(f FieldReportFn) {
	f("@loglov3-otl", SousGraphiteConfigV1)
	gcm.CallerInfo.EachField(f)
	if gcm.cfg == nil {
		f("sous-successful-connection", false)
		return
	}
	f("sous-successful-connection", true)
	f("graphite-server-address", gcm.cfg.Addr.String())
	f("graphite-flush-interval", int64(gcm.cfg.FlushInterval/time.Microsecond))
}
