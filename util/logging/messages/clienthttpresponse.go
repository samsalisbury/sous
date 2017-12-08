package messages

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opentable/sous/util/logging"
)

type httpLogEntry struct {
	logging.CallerInfo
	logging.Level
	serverSide     bool
	resourceFamily string
	method         string
	url            string
	server         string
	path           string
	parms          string
	status         int
	requestSize    int64
	responseSize   int64
	dur            time.Duration
}

/*
// ReportClientHTTPResponse reports a response recieved by Sous as a client, as provided as fields.
func ReportClientHTTPResponseFields(logger logging.LogSink, method, server, path string, parms map[string]string, status int, dur time.Duration) {
	m := newHTTPLogEntry(method, server, path, parms, status, dur)
	logging.Deliver(m, logger)
}
*/

// ReportClientHTTPResponse reports a response recieved by Sous as a client.
func ReportClientHTTPResponse(logger logging.LogSink, rz *http.Response, resName string, dur time.Duration) {
	m := buildHTTPLogMessage(false, rz.Request, rz.StatusCode, rz.ContentLength, resName, dur)
	m.ExcludeMe()
	logging.Deliver(m, logger)
}

// ReportServerHTTPResponse reports a response recieved by Sous as a client.
// n.b. this interface subject to change
func ReportServerHTTPResponse(logger logging.LogSink, rq *http.Request, statusCode int, contentLength int64, resName string, dur time.Duration) {
	m := buildHTTPLogMessage(true, rq, statusCode, contentLength, resName, dur)
	m.ExcludeMe()
	logging.Deliver(m, logger)
}

func buildHTTPLogMessage(server bool, rq *http.Request, statusCode int, responseContentLength int64, resName string, dur time.Duration) *httpLogEntry {
	url := rq.URL

	qps := map[string]string{}
	for k, v := range url.Query() {
		qps[k] = strings.Join(v, ",")
	}

	m := newHTTPLogEntry(
		server,
		resName,
		rq.Method,
		url.String(),
		statusCode,
		rq.ContentLength,
		responseContentLength,
		dur,
	)
	m.ExcludeMe()
	return m
}

func newHTTPLogEntry(server bool, resName, method, urlstring string, status int, rqSize, rzSize int64, dur time.Duration) *httpLogEntry {
	u, err := url.Parse(urlstring)
	if err != nil {
		u = &url.URL{}
	}

	return &httpLogEntry{
		Level:      logging.InformationLevel,
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),

		serverSide:     server,
		resourceFamily: resName,
		method:         method,
		url:            urlstring,
		server:         u.Host,
		path:           u.Path,
		parms:          u.RawQuery,
		status:         status,
		requestSize:    rqSize,
		responseSize:   rzSize,
		dur:            dur,
	}
}

func (msg *httpLogEntry) MetricsTo(metrics logging.MetricsSink) {
	side := "client"
	if msg.serverSide {
		side = "server"
	}
	metrics.UpdateTimer(fmt.Sprintf("%s-http-request-duration", side), msg.dur)
	metrics.UpdateTimer(fmt.Sprintf("%s-http-request-duration.%s", side, msg.resourceFamily), msg.dur)

	metrics.IncCounter(fmt.Sprintf("%s-http-status.%d", side, msg.status), 1)
	metrics.IncCounter(fmt.Sprintf("%s-http-status.%s.%d", side, msg.resourceFamily, msg.status), 1)

	metrics.UpdateSample(fmt.Sprintf("%s-http-request-size", side), msg.requestSize)
	metrics.UpdateSample(fmt.Sprintf("%s-http-request-size.%s", side, msg.resourceFamily), msg.requestSize)
}

func (msg *httpLogEntry) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-http-v1")
	f("resource-family", msg.resourceFamily)
	f("incoming", msg.serverSide)
	f("method", msg.method)
	f("status", msg.status)
	f("duration", int64(msg.dur/time.Microsecond))

	f("body-size", msg.requestSize)
	// body?
	f("response-size", msg.responseSize)
	// response-body?

	f("url", msg.url)
	f("url-hostname", msg.server)
	f("url-pathname", msg.path)
	f("url-querystring", msg.parms)
	msg.CallerInfo.EachField(f)
}

func (msg *httpLogEntry) Message() string {
	return fmt.Sprintf("%d", msg.status)
}
