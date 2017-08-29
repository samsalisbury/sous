package messages

import (
	"fmt"
	"net/http"
	"time"

	"github.com/opentable/sous/util/logging"
)

type clientHTTPResponse struct {
	logging.CallerInfo
	logging.CallTime
	logging.Level
	method string
	server string
	path   string
	parms  map[string]string
	status int
	dur    time.Duration
}

// ReportClientHTTPResponse reports a response recieved by Sous as a client, as provided as fields.
func ReportClientHTTPResponseFields(logger logging.LogSink, method, server, path string, parms map[string]string, status int, dur time.Duration) {
	m := newClientHTTPResponse(method, server, path, parms, status, dur)
	logging.Deliver(m, logger)
}

// ReportClientHTTPResponse reports a response recieved by Sous as a client.
func ReportClientHTTPResponse(logger logging.LogSink, rz http.Response, dur time.Duration) {
	url := rz.Request.URL

	m := newClientHTTPResponse(
		rz.Request.Method,
		url.Scheme+"://"+url.Host,
		url.Path,
		url.Query(),
		rz.StatusCode,
		dur,
	)
	logging.Deliver(m, logger)
}

func newClientHTTPResponse(method, server, path string, parms map[string]string, status int, dur time.Duration) *clientHTTPResponse {
	return &clientHTTPResponse{
		Level:      logging.InformationLevel,
		CallerInfo: logging.GetCallerInfo(),
		CallTime:   logging.GetCallTime(),

		method: method,
		server: server,
		path:   path,
		parms:  parms,
		status: status,
		dur:    dur,
	}
}

func (msg *clientHTTPResponse) MetricsTo(metrics logging.LogSink) {
	metrics.UpdateTimer("http-client", msg.dur)
}

func (msg *clientHTTPResponse) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-client-http-response-v1")
	f("method", msg.method)
	f("server", msg.server)
	f("path", msg.path)
	f("parms", msg.parms)
	f("status", msg.status)
	f("dur", msg.dur)
	msg.CallTime.EachField(f)
	msg.CallerInfo.EachField(f)
}

func (msg *clientHTTPResponse) Message() string {
	return fmt.Sprintf("%d", msg.status)
}
