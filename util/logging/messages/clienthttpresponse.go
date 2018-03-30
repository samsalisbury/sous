package messages

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/opentable/sous/util/logging"
)

// HTTPLogEntry struct to hold log entry messages
type HTTPLogEntry struct {
	logging.CallerInfo
	logging.Level
	message        string
	serverSide     bool
	isResponse     bool
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

var httpLogFile = ioutil.Discard

func init() {
	if test, debug := flag.CommandLine.Lookup("test.count"), os.Getenv("SOUS_DEBUG_SERVER"); test != nil || debug != "" {
		f, err := ioutil.TempFile("", "http-log")
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "Logging HTTP bodies to %q\n", f.Name())
		httpLogFile = f
	}
}

// ReportClientHTTPRequest reports a response recieved by Sous as a client.
// n.b. this interface subject to change
func ReportClientHTTPRequest(logger logging.LogSink, message string, rq *http.Request, resName string) {
	m := buildHTTPLogMessage(false, false, message, rq, 0, 0, resName, 0)
	m.ExcludeMe()
	logging.Deliver(m, logger)
}

// ReportClientHTTPResponse reports a response recieved by Sous as a client.
func ReportClientHTTPResponse(logger logging.LogSink, message string, rz *http.Response, resName string, dur time.Duration) {
	fmt.Fprintf(httpLogFile, "\n\n=== %s %s -> %s ===\n\n", rz.Request.Method, rz.Request.URL.String(), rz.Status)
	if rz.Body != nil {
		b := &bytes.Buffer{}
		tee := io.TeeReader(rz.Body, b)
		io.Copy(httpLogFile, tee)
		rz.Body = ioutil.NopCloser(b)
	}

	m := buildHTTPLogMessage(false, true, message, rz.Request, rz.StatusCode, rz.ContentLength, resName, dur)
	m.ExcludeMe()
	logging.Deliver(m, logger)
}

// ReportServerHTTPRequest reports a response recieved by Sous as a client.
// n.b. this interface subject to change
func ReportServerHTTPRequest(logger logging.LogSink, message string, rq *http.Request, resName string) {
	m := buildHTTPLogMessage(true, false, message, rq, 0, 0, resName, 0)
	m.ExcludeMe()
	logging.Deliver(m, logger)
}

// ReportServerHTTPResponse reports a response recieved by Sous as a client.
func ReportServerHTTPResponse(logger logging.LogSink, message string, rz *http.Response, resName string, dur time.Duration) {
	m := buildHTTPLogMessage(true, true, message, rz.Request, rz.StatusCode, rz.ContentLength, resName, dur)
	m.ExcludeMe()
	logging.Deliver(m, logger)
}

// ReportServerHTTPResponding reports a response to a request - this is useful in cases where a ResponseWriter is encapsulating the actual response.
func ReportServerHTTPResponding(logger logging.LogSink, message string, req *http.Request, status int, responseContentLength int64, resName string, dur time.Duration) {
	m := buildHTTPLogMessage(true, true, message, req, status, responseContentLength, resName, dur)
	m.ExcludeMe()
	logging.Deliver(m, logger)
}

func buildHTTPLogMessage(
	server, response bool,
	message string,
	rq *http.Request,
	statusCode int,
	responseContentLength int64,
	resName string,
	dur time.Duration,
) *HTTPLogEntry {
	url := rq.URL

	qps := map[string]string{}
	for k, v := range url.Query() {
		qps[k] = strings.Join(v, ",")
	}

	m := newHTTPLogEntry(
		message,
		server,
		response,
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

func newHTTPLogEntry(
	message string,
	server, response bool,
	resName, method, urlstring string,
	status int,
	rqSize, rzSize int64,
	dur time.Duration,
) *HTTPLogEntry {
	u, err := url.Parse(urlstring)
	if err != nil {
		u = &url.URL{}
	}

	lvl := logging.InformationLevel
	if status < 400 {
		lvl = logging.ExtraDebug1Level
	}

	return &HTTPLogEntry{
		Level:      lvl,
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),

		message:        message,
		serverSide:     server,
		isResponse:     response,
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
	// Could be simpler, but this is a precursor to logging the "isResponse" field
	incoming := (msg.serverSide && !msg.isResponse) || (!msg.serverSide && msg.isResponse)
	f("@loglov3-otl", "sous-http-v1")
	f("resource-family", msg.resourceFamily)
	f("incoming", incoming)
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
	var channelName string
	if msg.serverSide {
		if msg.isResponse {
			channelName = "<- Server "
		} else {
			channelName = "-> Server "
		}
	} else {
		if msg.isResponse {
			channelName = "Client <- "
		} else {
			channelName = "Client -> "
		}
	}
	return channelName + msg.message
}
