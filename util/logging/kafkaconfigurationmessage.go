package logging

import (
	"github.com/tracer0tong/kafkalogrus"
)

type kafkaConfigurationMessage struct {
	CallerInfo
	hook    *kafkalogrus.KafkaLogrusHook
	brokers []string
	topic   string
}

func reportKafkaConfig(hook *kafkalogrus.KafkaLogrusHook, cfg Config, ls LogSink) {
	msg := kafkaConfigurationMessage{
		CallerInfo: GetCallerInfo(),
		hook:       hook,
		brokers:    cfg.getBrokers(),
		topic:      cfg.Kafka.Topic,
	}
	Deliver(msg, ls)
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
	f("@loglov3-otl", "sous-kafka-config-v1")
	kcm.CallerInfo.EachField(f)
	if kcm.hook == nil {
		f("sous-successful-connection", false)
		return
	}
	f("sous-successful-connection", true)
	f("kafka-logging-topic", kcm.topic)
	f("kafka-brokers", kcm.brokers)
	f("kafka-logger-id", kcm.hook.Id())
	f("kafka-logging-levels", kcm.hook.Levels())
}
