package messages

import (
	"net/http"
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

// This test demonstrates how to use AssertReport and AssertMessageFields separately,
// along with why you might want to.
func TestReportClientHTTPResponseFields_assertReport(t *testing.T) {
	res := buildHTTPResponse(t, "GET", "http://example.com/api?a=a", 200, 0, 123)
	control, message := logging.AssertReport(t, func(logger logging.LogSink) {
		ReportClientHTTPResponse(logger, "test", res, "example-api", time.Millisecond*30)
	})

	assert.Equal(t, control.CallsTo("LogMessage")[0].PassedArgs().Get(0), logging.ExtraDebug1Level)
	assert.Len(t, control.Metrics.CallsTo("UpdateTimer"), 3)
	assert.Len(t, control.Metrics.CallsTo("UpdateSample"), 6)
	assert.Len(t, control.Metrics.CallsTo("IncCounter"), 3)

	logging.AssertMessageFields(t, message,
		logging.StandardVariableFields,
		map[string]interface{}{
			"@loglov3-otl":    "sous-http-v1",
			"incoming":        true,
			"resource-family": "example-api",
			"method":          "GET",
			"url":             "http://example.com/api?a=a",
			"url-hostname":    "example.com",
			"url-pathname":    "/api",
			"url-querystring": "a=a",
			"duration":        int64(30000),
			"body-size":       int64(0),
			"response-size":   int64(123),
			"status":          200,
		})
}

func TestReportClientHTTPResponseFields_InfoLevelOnErrors(t *testing.T) {
	res := buildHTTPResponse(t, "GET", "http://example.com/api?a=a", 401, 0, 123)
	control, _ := logging.AssertReport(t, func(logger logging.LogSink) {
		ReportClientHTTPResponse(logger, "test", res, "example-api", time.Millisecond*30)
	})

	assert.Equal(t, control.CallsTo("LogMessage")[0].PassedArgs().Get(0), logging.InformationLevel)
}

func TestReportClientHTTPRequestFields(t *testing.T) {
	req := buildHTTPRequest(t, "GET", "http://example.com/api?a=a", 0)
	logging.AssertReportFields(t,
		func(logger logging.LogSink) {
			ReportClientHTTPRequest(logger, "test", req, "example-api")
		},
		logging.StandardVariableFields,
		map[string]interface{}{
			"@loglov3-otl":    "sous-http-v1",
			"incoming":        false,
			"resource-family": "example-api",
			"method":          "GET",
			"url":             "http://example.com/api?a=a",
			"url-hostname":    "example.com",
			"url-pathname":    "/api",
			"url-querystring": "a=a",
			"duration":        int64(0),
			"body-size":       int64(0),
			"response-size":   int64(0),
			"status":          0,
		})
}

func TestReportServerHTTPRequestFields(t *testing.T) {
	req := buildHTTPRequest(t, "GET", "http://example.com/api?a=a", 0)
	logging.AssertReportFields(t,
		func(logger logging.LogSink) {
			ReportServerHTTPRequest(logger, "test", req, "example-api")
		},
		logging.StandardVariableFields,
		map[string]interface{}{
			"@loglov3-otl":    "sous-http-v1",
			"incoming":        true,
			"resource-family": "example-api",
			"method":          "GET",
			"url":             "http://example.com/api?a=a",
			"url-hostname":    "example.com",
			"url-pathname":    "/api",
			"url-querystring": "a=a",
			"duration":        int64(0),
			"body-size":       int64(0),
			"response-size":   int64(0),
			"status":          0,
		})
}

func TestReportClientHTTPResponseFields(t *testing.T) {
	res := buildHTTPResponse(t, "GET", "http://example.com/api?a=a", 200, 0, 123)
	logging.AssertReportFields(t,
		func(logger logging.LogSink) {
			ReportClientHTTPResponse(logger, "test", res, "example-api", time.Millisecond*30)
		},
		logging.StandardVariableFields,
		map[string]interface{}{
			"@loglov3-otl":    "sous-http-v1",
			"incoming":        true,
			"resource-family": "example-api",
			"method":          "GET",
			"url":             "http://example.com/api?a=a",
			"url-hostname":    "example.com",
			"url-pathname":    "/api",
			"url-querystring": "a=a",
			"duration":        int64(30000),
			"body-size":       int64(0),
			"response-size":   int64(123),
			"status":          200,
		})
}

func TestReportServerHTTPResponseFields(t *testing.T) {
	res := buildHTTPResponse(t, "GET", "http://example.com/api?a=a", 200, 0, 123)
	logging.AssertReportFields(t,
		func(logger logging.LogSink) {
			ReportServerHTTPResponse(logger, "test", res, "example-api", time.Millisecond*30)
		},
		logging.StandardVariableFields,
		map[string]interface{}{
			"@loglov3-otl":    "sous-http-v1",
			"incoming":        false,
			"resource-family": "example-api",
			"method":          "GET",
			"url":             "http://example.com/api?a=a",
			"url-hostname":    "example.com",
			"url-pathname":    "/api",
			"url-querystring": "a=a",
			"duration":        int64(30000),
			"body-size":       int64(0),
			"response-size":   int64(123),
			"status":          200,
		})
}

func TestReportServerHTTPRespondingFields(t *testing.T) {
	req := buildHTTPRequest(t, "PUT", "http://example.com/api?a=a", 20)
	logging.AssertReportFields(t,
		func(logger logging.LogSink) {
			ReportServerHTTPResponding(logger, "test", req, 200, 123, "example-api", 30000)
		},
		logging.StandardVariableFields,
		map[string]interface{}{
			"@loglov3-otl":    "sous-http-v1",
			"incoming":        false,
			"resource-family": "example-api",
			"method":          "PUT",
			"url":             "http://example.com/api?a=a",
			"url-hostname":    "example.com",
			"url-pathname":    "/api",
			"url-querystring": "a=a",
			"duration":        int64(30),
			"body-size":       int64(20),
			"response-size":   int64(123),
			"status":          200,
		})
}

func buildHTTPRequest(t *testing.T, method string, url string, rqLength int64) *http.Request {
	t.Helper()
	rq, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("error building dummy request: %v", err)
	}
	rq.ContentLength = rqLength
	return rq
}

func buildHTTPResponse(t *testing.T, method string, url string, status int, rqLength int64, rzLength int64) *http.Response {
	t.Helper()
	rq := buildHTTPRequest(t, method, url, rqLength)

	return &http.Response{
		Request:       rq,
		StatusCode:    status,
		ContentLength: rzLength,
	}
}
