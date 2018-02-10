package messages

import (
	"testing"

	"github.com/opentable/sous/util/logging"
)

func TestReportLogFieldsMessage_Complete(t *testing.T) {
	logging.AssertReportFields(t,
		func(ls logging.LogSink) {
			cfg := logging.Config{}
			cfg.Kafka.Topic = "test-topic"
			cfg.Kafka.BrokerList = "broker1,broker2,broker3"

			ReportLogFieldsMessage("This is test message", logging.DebugLevel, ls, cfg)
		},
		logging.StandardVariableFields,
		map[string]interface{}{
			"fields":       "Basic,Kafka,Graphite,Config,Level,DisableConsole,Enabled,DefaultLevel,Topic,Brokers,BrokerList,Server",
			"types":        "Config,string,bool",
			"jsonStruct":   "{\"message\":{\"array\":[\"{\\\"Basic\\\":{\\\"DisableConsole\\\":false,\\\"Level\\\":\\\"\\\"},\\\"Graphite\\\":{\\\"Enabled\\\":false,\\\"Server\\\":\\\"\\\"},\\\"Kafka\\\":{\\\"BrokerList\\\":\\\"broker1,broker2,broker3\\\",\\\"Brokers\\\":null,\\\"DefaultLevel\\\":\\\"\\\",\\\"Enabled\\\":false,\\\"Topic\\\":\\\"test-topic\\\"}}\"]}}",
			"@loglov3-otl": "sous-generic-v1",
		})
}

func TestReportLogFieldsMessage_NoInterface(t *testing.T) {
	logging.AssertReportFields(t,
		func(ls logging.LogSink) {
			ReportLogFieldsMessage("This is test message no interface", logging.DebugLevel, ls)
		},
		logging.StandardVariableFields,
		map[string]interface{}{
			"fields":       "",
			"types":        "",
			"jsonStruct":   "{\"message\":{\"array\":[]}}",
			"@loglov3-otl": "sous-generic-v1",
		})
}
func TestReportLogFieldsMessage_String(t *testing.T) {
	logging.AssertReportFields(t,
		func(ls logging.LogSink) {
			ReportLogFieldsMessage("This is test message passing just a string", logging.DebugLevel, ls, "simple string")
		},
		logging.StandardVariableFields,
		map[string]interface{}{
			"fields":       "",
			"types":        "string",
			"jsonStruct":   "{\"message\":{\"array\":[\"{\\\"string\\\":{\\\"string\\\":\\\"simple string\\\"}}\"]}}",
			"@loglov3-otl": "sous-generic-v1",
		})
}

func TestReportLogFieldsMessage_StructAndString(t *testing.T) {
	logging.AssertReportFields(t,
		func(ls logging.LogSink) {
			cfg := logging.Config{}
			cfg.Kafka.Topic = "test-topic"
			cfg.Kafka.BrokerList = "broker1,broker2,broker3"

			ReportLogFieldsMessage("This is test message", logging.DebugLevel, ls, cfg, "simple string")
		},
		logging.StandardVariableFields,
		map[string]interface{}{
			"fields":       "Basic,Kafka,Graphite,Config,Level,DisableConsole,Enabled,DefaultLevel,Topic,Brokers,BrokerList,Server",
			"types":        "Config,string,bool",
			"jsonStruct":   "{\"message\":{\"array\":[\"{\\\"Basic\\\":{\\\"DisableConsole\\\":false,\\\"Level\\\":\\\"\\\"},\\\"Graphite\\\":{\\\"Enabled\\\":false,\\\"Server\\\":\\\"\\\"},\\\"Kafka\\\":{\\\"BrokerList\\\":\\\"broker1,broker2,broker3\\\",\\\"Brokers\\\":null,\\\"DefaultLevel\\\":\\\"\\\",\\\"Enabled\\\":false,\\\"Topic\\\":\\\"test-topic\\\"}}\",\"{\\\"string\\\":{\\\"string\\\":\\\"simple string\\\"}}\"]}}",
			"@loglov3-otl": "sous-generic-v1",
		})
}
