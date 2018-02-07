package messages

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opentable/sous/util/logging"
)

// HTTPLogEntry struct to hold log entry messages
type HTTPLogEntry struct {
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

// ReportClientHTTPResponse reports a response recieved by Sous as a client.
func ReportClientHTTPResponse(logger logging.LogSink, rz *http.Response, resName string, dur time.Duration) {
	// XXX dur should in fact be "start time.Time" and duration be computed here.
	// swaggering now depends on this, so it's more of a hassle.
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

// BuildClientHTTPResponse reports a response recieved by Sous as a client.
func BuildClientHTTPResponse(rz *http.Response, resName string, dur time.Duration) *HTTPLogEntry {
	// XXX dur should in fact be "start time.Time" and duration be computed here.
	// swaggering now depends on this, so it's more of a hassle.
	m := buildHTTPLogMessage(false, rz.Request, rz.StatusCode, rz.ContentLength, resName, dur)
	m.ExcludeMe()
	return m
}

// BuildServerHTTPResponse reports a response recieved by Sous as a client.
// n.b. this interface subject to change
func BuildServerHTTPResponse(rq *http.Request, statusCode int, contentLength int64, resName string, dur time.Duration) *HTTPLogEntry {
	m := buildHTTPLogMessage(true, rq, statusCode, contentLength, resName, dur)
	m.ExcludeMe()
	return m
}

func buildHTTPLogMessage(server bool, rq *http.Request, statusCode int, responseContentLength int64, resName string, dur time.Duration) *HTTPLogEntry {
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
	m.ExcludePathPattern("github.com/opentable/swaggering")     // XXX should be more local
	m.ExcludePathPattern("github.com/opentable/go-singularity") // XXX should be more local
	return m
}

func newHTTPLogEntry(server bool, resName, method, urlstring string, status int, rqSize, rzSize int64, dur time.Duration) *HTTPLogEntry {
	u, err := url.Parse(urlstring)
	if err != nil {
		u = &url.URL{}
	}

	return &HTTPLogEntry{
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

// MetricsTo function to send metrics to graphite
func (msg *HTTPLogEntry) MetricsTo(metrics logging.MetricsSink) {
	side := "client"
	if msg.serverSide {
		side = "server"
	}
	host := strings.Replace(msg.server, ".", "_", -1)

	nameGen := func(pat string, fields ...interface{}) []string {
		base := fmt.Sprintf("%s-%s-%s", side, msg.method, pat)
		if len(fields) > 0 {
			base = fmt.Sprintf(base, fields...)
		}

		// put the metrics into 3 buckets:
		// by "resource family" - a way of aggregating URLs
		// by hostname (with s/./_/g for graphite's comfort)
		// by the pair of (resource, host)
		return []string{
			fmt.Sprintf("%s.%s", base, msg.resourceFamily),
			fmt.Sprintf("%s.%s", base, host),
			fmt.Sprintf("%s.%s.%s", base, host, msg.resourceFamily),
		}
	}

	for _, name := range nameGen("http-request-duration") {
		metrics.UpdateTimer(name, msg.dur)
	}
	for _, name := range nameGen("http-status.%d", msg.status) {
		metrics.IncCounter(name, 1)
	}

	for _, name := range nameGen("http-request-size") {
		metrics.UpdateSample(name, msg.requestSize)
	}

	for _, name := range nameGen("http-response-size") {
		metrics.UpdateSample(name, msg.responseSize)
	}
}

// EachField to populate the proper fields from message
func (msg *HTTPLogEntry) EachField(f logging.FieldReportFn) {
	msg.EachFieldWithoutCallerInfo(f)
	msg.CallerInfo.EachField(f)
}

// EachFieldWithoutCallerInfo allows sub messages to populate
// logging.FieldReportFn without out having to call CallerInfo
func (msg *HTTPLogEntry) EachFieldWithoutCallerInfo(f logging.FieldReportFn) {
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
}

// Status method to retrieve status from message
func (msg *HTTPLogEntry) Status() int {
	return msg.status
}

// Message retrieve message from log message
func (msg *HTTPLogEntry) Message() string {
	return fmt.Sprintf("%d", msg.status)
}
