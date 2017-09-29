package logging

import graphite "github.com/cyberdelia/go-metrics-graphite"

type graphiteConfigMessage struct {
	CallerInfo
	cfg *graphite.Config
}

func reportGraphiteConfig(cfg *graphite.Config, ls LogSink) {
	msg := graphiteConfigMessage{
		CallerInfo: GetCallerInfo(),
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
	f("@loglov3-otl", "sous-graphite-config")
	gcm.CallerInfo.EachField(f)
	if gcm.cfg == nil {
		return
	}
	f("server-addr", gcm.cfg.Addr)
	f("flush-interval", gcm.cfg.FlushInterval)
}
