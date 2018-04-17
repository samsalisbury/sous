package logging

import (
	"testing"
)

func TestReportKafkaConfiguration_Zero(t *testing.T) {
	var hook *liveKafkaSink
	var cfg Config

	reportKafkaConfig(hook, cfg, SilentLogSet())
	AssertReportFields(t,
		func(ls LogSink) {
			reportKafkaConfig(hook, cfg, ls)
		},
		StandardVariableFields,
		map[string]interface{}{
			"@loglov3-otl":               SousKafkaConfigV1,
			"severity":                   WarningLevel,
			"call-stack-message":         "Not connecting to Kafka.",
			"sous-successful-connection": false,
			"kafka-logging-topic":        "",
			"kafka-brokers":              "",
			"kafka-logger-id":            "",
		})
}

func TestReportKafkaConfiguration_Complete(t *testing.T) {
	AssertReportFields(t,
		func(ls LogSink) {
			hook := &liveKafkaSink{}
			cfg := Config{}
			cfg.Kafka.Topic = "test-topic"
			cfg.Kafka.BrokerList = "broker1,broker2,broker3"

			reportKafkaConfig(hook, cfg, ls)
		},
		StandardVariableFields,
		map[string]interface{}{
			"@loglov3-otl":               SousKafkaConfigV1,
			"severity":                   InformationLevel,
			"call-stack-message":         "Connecting to Kafka",
			"kafka-logging-topic":        "test-topic",
			"kafka-brokers":              "broker1,broker2,broker3",
			"kafka-logger-id":            "",
			"sous-successful-connection": true,
		})
}
