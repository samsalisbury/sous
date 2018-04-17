package logging

import (
	"strings"
)

func reportKafkaConfig(hook kafkaSink, cfg Config, ls LogSink) {
	msg := ConsoleAndMessage("Not connecting to Kafka.")
	lvl := WarningLevel
	succ := false
	id := ""
	if hook != nil && hook.live() {
		msg = ConsoleAndMessage("Connecting to Kafka")
		lvl = InformationLevel
		succ = true
		id = hook.id()
	}

	Deliver(ls,
		SousKafkaConfigV1,
		GetCallerInfo(NotHere()),
		lvl,
		msg,
		KV(SousSuccessfulConnection, succ),
		KV(KafkaLoggingTopic, cfg.Kafka.Topic),
		KV(KafkaBrokers, strings.Join(cfg.getBrokers(), ",")),
		KV(KafkaLoggerId, id),
	)
}
