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
			"jsonStruct":   "{\"message\":\"{\\\"Basic\\\":{\\\"DisableConsole\\\":false,\\\"Level\\\":\\\"\\\"},\\\"Graphite\\\":{\\\"Enabled\\\":false,\\\"Server\\\":\\\"\\\"},\\\"Kafka\\\":{\\\"BrokerList\\\":\\\"broker1,broker2,broker3\\\",\\\"Brokers\\\":null,\\\"DefaultLevel\\\":\\\"\\\",\\\"Enabled\\\":false,\\\"Topic\\\":\\\"test-topic\\\"}}\"}",
			"@loglov3-otl": "sous-generic-v1",
		})
}
