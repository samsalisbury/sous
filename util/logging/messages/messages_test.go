package messages

import (
	"net/http"
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func returnTestMap(callStack string) map[string]interface{} {
	return map[string]interface{}{
		"@loglov3-otl":        "sous-http-v1",
		"incoming":            false,
		"resource-family":     "example-api",
		"method":              "GET",
		"url":                 "http://example.com/api?a=a",
		"url-hostname":        "example.com",
		"url-pathname":        "/api",
		"url-querystring":     "a=a",
		"duration":            int64(30000),
		"body-size":           int64(0),
		"response-size":       int64(123),
		"status":              200,
		"call-stack-function": callStack,
	}
}
func TestCHResponseFields(t *testing.T) {
	res := buildHTTPResponse(t, "GET", "http://example.com/api?a=a", 200, 0, 123)
	control, message := logging.AssertReport(t, func(logger logging.LogSink) {
		ReportClientHTTPResponse(logger, res, "example-api", time.Millisecond*30)
	})

	assert.Equal(t, control.CallsTo("LogMessage")[0].PassedArgs().Get(0), logging.InformationLevel)
	assert.Len(t, control.Metrics.CallsTo("UpdateTimer"), 3)
	assert.Len(t, control.Metrics.CallsTo("UpdateSample"), 6)
	assert.Len(t, control.Metrics.CallsTo("IncCounter"), 3)

	fieldsMap := returnTestMap("github.com/opentable/sous/util/logging/messages.TestCHResponseFields")
	logging.AssertMessageFields(t, message,
		logging.StandardVariableFields, fieldsMap,
	)
}

// This test mostly exists as a demo of AssertReportFields
func TestReportCHResponseFields(t *testing.T) {
	res := buildHTTPResponse(t, "GET", "http://example.com/api?a=a", 200, 0, 123)

	fieldsMap := returnTestMap("github.com/opentable/sous/util/logging/messages.TestReportCHResponseFields")
	logging.AssertReportFields(t,
		func(logger logging.LogSink) {
			ReportClientHTTPResponse(logger, res, "example-api", time.Millisecond*30)
		},
		logging.StandardVariableFields,
		fieldsMap)
}

func buildHTTPResponse(t *testing.T, method string, url string, status int, rqLength int64, rzLength int64) *http.Response {
	t.Helper()
	rq, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("error building dummy request: %v", err)
	}
	rq.ContentLength = rqLength

	return &http.Response{
		Request:       rq,
		StatusCode:    status,
		ContentLength: rzLength,
	}
}
