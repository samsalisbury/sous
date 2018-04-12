package logging

import (
	"strings"
)

type kafkaConfigurationMessage struct {
	CallerInfo
	hook    *kafkaSink
	brokers []string
	topic   string
}

func reportKafkaConfig(hook *kafkaSink, cfg Config, ls LogSink) {
	msg := kafkaConfigurationMessage{
		CallerInfo: GetCallerInfo(),
		hook:       hook,
		brokers:    cfg.getBrokers(),
		topic:      cfg.Kafka.Topic,
	}
	msg.ExcludeMe()

	if hook == nil {
		Deliver(ls,
			SousKafkaConfigV1,
			WarningLevel,
			GetCallerInfo(NotHere()),
			MessageField("Not connecting to Kafka."),
			KV(SousSuccessfulConnection, false),
		)
		return
	}

	Deliver(ls,
		SousKafkaConfigV1,
		InformationLevel,
		GetCallerInfo(NotHere()),
		ConsoleAndMessage("Connecting to Kafka"),
		KV(SousSuccessfulConnection, true),
		KV(KafkaLoggingTopic, cfg.Kafka.Topic),
		KV(KafkaBrokers, strings.Join(cfg.getBrokers(), ",")),
		KV(KafkaLoggerId, hook.ID()),
		KV(KafkaLoggingLevels, hook.level.String()),
	)
}
