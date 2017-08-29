package logging

import (
	"testing"
	"time"
)

func TestReportCHResponse(t *testing.T) {
	logger, spy := newLogSinkSpy()
	ReportClientHTTPResponse(logger, "http://example.com", "/api", map[string]string{"a": "a"}, 200, time.Millisecond*30)

}
