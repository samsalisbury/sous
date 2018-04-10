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
	Deliver(ls, msg)
}

func (kcm kafkaConfigurationMessage) DefaultLevel() Level {
	return InformationLevel
}

func (kcm kafkaConfigurationMessage) Message() string {
	if kcm.hook == nil {
		return "Not connecting to Kafka."
	}
	return "Connecting to Kafka"
}

func (kcm kafkaConfigurationMessage) EachField(f FieldReportFn) {
	f("@loglov3-otl", SousKafkaConfigV1)
	kcm.CallerInfo.EachField(f)
	if kcm.hook == nil {
		f("sous-successful-connection", false)
		return
	}
	f("sous-successful-connection", true)
	f("kafka-logging-topic", kcm.topic)
	f("kafka-brokers", strings.Join(kcm.brokers, ","))
	f("kafka-logger-id", kcm.hook.ID())
	f("kafka-logging-levels", kcm.hook.level.String())
}
