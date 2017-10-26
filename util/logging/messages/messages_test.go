package messages

import (
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportCHResponseFields(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()
	msg := newClientHTTPResponse("GET", "http://example.com/api?a=a", 200, 0, 123, time.Millisecond*30)
	logging.Deliver(msg, logger)

	assert.Len(t, control.Metrics.CallsTo("UpdateTimer"), 1)
	logCalls := control.CallsTo("LogMessage")
	require.Len(t, logCalls, 1)
	assert.Equal(t, logCalls[0].PassedArgs().Get(0), logging.InformationLevel)
	message := logCalls[0].PassedArgs().Get(1).(logging.LogMessage)

	/*
		"line":610,
		"function":"testing.tRunner",
		"file":"/nix/store/br0ngwcjyffc7d060spw44wah1hdnlwn-go-1.7.4/share/go/src/testing/testing.go",
		"time":logging.callTime{sec:63639633602, nsec:854240181, loc:(*time.Location)(0x8f3780)},
	*/

	fixedFields := []string{"call-stack-line-number", "call-stack-function", "call-stack-file", "@timestamp", "thread-name"}

	variableFields := map[string]interface{}{
		"@loglov3-otl":    "http-v1",
		"incoming":        false,
		"method":          "GET",
		"url":             "http://example.com/api?a=a",
		"url-hostname":    "example.com",
		"url-pathname":    "/api",
		"url-querystring": "a=a",
		"duration":        int64(30000),
		"body-size":       int64(0),
		"response-size":   int64(123),
		"status":          200,
	}

	logging.AssertMessageFields(t, message, fixedFields, variableFields)
}
