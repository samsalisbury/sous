package logging

import (
	"testing"

	"github.com/opentable/sous/util/logging/constants"
)

func TestReportKafkaConfiguration_Zero(t *testing.T) {
	AssertReportFields(t,
		func(ls LogSink) {
			var hook *kafkaSink
			var cfg Config

			reportKafkaConfig(hook, cfg, ls)
		},
		StandardVariableFields,
		map[string]interface{}{
			"@loglov3-otl":               constants.SousKafkaConfigV1,
			"sous-successful-connection": false,
		})
}

func TestReportKafkaConfiguration_Complete(t *testing.T) {
	AssertReportFields(t,
		func(ls LogSink) {
			hook := &kafkaSink{}
			cfg := Config{}
			cfg.Kafka.Topic = "test-topic"
			cfg.Kafka.BrokerList = "broker1,broker2,broker3"

			reportKafkaConfig(hook, cfg, ls)
		},
		StandardVariableFields,
		map[string]interface{}{
			"kafka-logging-topic":        "test-topic",
			"kafka-brokers":              "broker1,broker2,broker3",
			"kafka-logging-levels":       "CriticalLevel",
			"kafka-logger-id":            "",
			"@loglov3-otl":               constants.SousKafkaConfigV1,
			"sous-successful-connection": true,
		})
}
