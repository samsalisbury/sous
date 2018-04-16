package logging

import (
	"strings"
)

func reportKafkaConfig(hook kafkaSink, cfg Config, ls LogSink) {
	msg := ConsoleAndMessage("Connecting to Kafka")
	lvl := InformationLevel
	succ := true
	if !hook.live() {
		msg = ConsoleAndMessage("Not connecting to Kafka.")
		lvl = WarningLevel
		succ = false
	}

	Deliver(ls,
		SousKafkaConfigV1,
		GetCallerInfo(NotHere()),
		lvl,
		msg,
		KV(SousSuccessfulConnection, succ),
		KV(KafkaLoggingTopic, cfg.Kafka.Topic),
		KV(KafkaBrokers, strings.Join(cfg.getBrokers(), ",")),
		KV(KafkaLoggerId, hook.id()),
	)
}
