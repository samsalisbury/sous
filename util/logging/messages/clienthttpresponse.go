package messages

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opentable/sous/util/logging"
)

type clientHTTPResponse struct {
	logging.CallerInfo
	logging.Level
	method       string
	url          string
	server       string
	path         string
	parms        string
	status       int
	requestSize  int64
	responseSize int64
	dur          time.Duration
}

/*
// ReportClientHTTPResponse reports a response recieved by Sous as a client, as provided as fields.
func ReportClientHTTPResponseFields(logger logging.LogSink, method, server, path string, parms map[string]string, status int, dur time.Duration) {
	m := newClientHTTPResponse(method, server, path, parms, status, dur)
	logging.Deliver(m, logger)
}
*/

// ReportClientHTTPResponse reports a response recieved by Sous as a client.
func ReportClientHTTPResponse(logger logging.LogSink, rz http.Response, dur time.Duration) {
	url := rz.Request.URL

	qps := map[string]string{}
	for k, v := range url.Query() {
		qps[k] = strings.Join(v, ",")
	}

	m := newClientHTTPResponse(
		rz.Request.Method,
		url.String(),
		rz.StatusCode,
		rz.Request.ContentLength,
		rz.ContentLength,
		dur,
	)
	logging.Deliver(m, logger)
}

func newClientHTTPResponse(method, urlstring string, status int, rqSize, rzSize int64, dur time.Duration) *clientHTTPResponse {
	u, err := url.Parse(urlstring)
	if err != nil {
		u = &url.URL{}
	}

	return &clientHTTPResponse{
		Level:      logging.InformationLevel,
		CallerInfo: logging.GetCallerInfo(),

		method:       method,
		url:          urlstring,
		server:       u.Host,
		path:         u.Path,
		parms:        u.RawQuery,
		status:       status,
		requestSize:  rqSize,
		responseSize: rzSize,
		dur:          dur,
	}
}

func (msg *clientHTTPResponse) MetricsTo(metrics logging.MetricsSink) {
	metrics.UpdateTimer("http-request-duration", msg.dur)
}

func (msg *clientHTTPResponse) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "http-v1")
	f("incoming", false)
	f("method", msg.method)
	f("status", msg.status)
	f("duration", msg.dur)

	f("url", msg.url)
	f("body-size", msg.requestSize)
	f("response-size", msg.responseSize)

	f("server", msg.server)
	f("path", msg.path)
	f("querystring", msg.parms)
	msg.CallerInfo.EachField(f)
}

func (msg *clientHTTPResponse) Message() string {
	return fmt.Sprintf("%d", msg.status)
}
