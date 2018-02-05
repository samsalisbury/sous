package logging

import (
	"fmt"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/nyarly/spies"
	"github.com/opentable/sous/util/yaml"
	"github.com/stretchr/testify/assert"
)

type (
	metricsSinkSpy struct {
		spy *spies.Spy
	}

	metricsSinkController struct {
		*spies.Spy
	}

	writeDonerSpy struct {
		spy *spies.Spy
	}

	writeDonerController struct {
		*spies.Spy
	}

	logSinkSpy struct {
		spy *spies.Spy
	}

	// LogSinkController allows testing code to manipulate and inspect the spies
	// returned by NewLogSinkSpy
	LogSinkController struct {
		*spies.Spy
		Metrics metricsSinkController
		Console writeDonerController
	}
)

// NewLogSinkSpy returns a spy/controller pair for testing purposes.
// (see LogSet for a general purpose implementation of LogSink)
func NewLogSinkSpy() (LogSink, LogSinkController) {
	spy := spies.NewSpy()

	console, cc := NewWriteDonerSpy()
	metrics, mc := NewMetricsSpy()

	ctrl := LogSinkController{
		Spy:     spy,
		Metrics: mc,
		Console: cc,
	}
	ctrl.MatchMethod("Console", spies.AnyArgs, console)
	ctrl.MatchMethod("Metrics", spies.AnyArgs, metrics)

	return logSinkSpy{spy: spy}, ctrl
}

func (lss logSinkSpy) LogMessage(lvl Level, msg LogMessage) {
	lss.spy.Called(lvl, msg)
}

// These do what LogSet does so that it'll be easier to replace the interface
func (lss logSinkSpy) Vomitf(f string, as ...interface{}) {
	m := NewGenericMsg(ExtraDebug1Level, fmt.Sprintf(f, as...), nil, false)
	Deliver(m, lss)
}

func (lss logSinkSpy) Debugf(f string, as ...interface{}) {
	m := NewGenericMsg(DebugLevel, fmt.Sprintf(f, as...), nil, false)
	Deliver(m, lss)
}

func (lss logSinkSpy) Warnf(f string, as ...interface{}) {
	m := NewGenericMsg(WarningLevel, fmt.Sprintf(f, as...), nil, false)
	Deliver(m, lss)
}

func (lss logSinkSpy) Child(name string) LogSink {
	lss.spy.Called(name)
	return lss //easier than managing a whole new lss
}

func (lss logSinkSpy) Console() WriteDoner {
	res := lss.spy.Called()
	return res.Get(0).(WriteDoner)
}

func (lss logSinkSpy) Metrics() MetricsSink {
	res := lss.spy.Called()
	return res.Get(0).(MetricsSink)
}

func (lss logSinkSpy) AtExit() {
	lss.spy.Called()
}

// Returns a spy/controller pair
func NewMetricsSpy() (MetricsSink, metricsSinkController) {
	spy := spies.NewSpy()
	return metricsSinkSpy{spy}, metricsSinkController{spy}
}

func (mss metricsSinkSpy) ClearCounter(name string) {
	mss.spy.Called(name)
}

func (mss metricsSinkSpy) IncCounter(name string, amount int64) {
	mss.spy.Called(name, amount)
}

func (mss metricsSinkSpy) DecCounter(name string, amount int64) {
	mss.spy.Called(name, amount)
}

func (mss metricsSinkSpy) UpdateTimer(name string, dur time.Duration) {
	mss.spy.Called(name, dur)
}

func (mss metricsSinkSpy) UpdateTimerSince(name string, time time.Time) {
	mss.spy.Called(name, time)
}

func (mss metricsSinkSpy) UpdateSample(name string, value int64) {
	mss.spy.Called(name, value)
}

func (mss metricsSinkSpy) Done() {
	mss.spy.Called()
}

// NewWriteDonerSpy returns a spy/controller pair for WriteDoner
func NewWriteDonerSpy() (WriteDoner, writeDonerController) {
	spy := spies.NewSpy()
	return writeDonerSpy{spy: spy}, writeDonerController{Spy: spy}
}

func (wds writeDonerSpy) Write(p []byte) (n int, err error) {
	res := wds.spy.Called()
	return res.Int(0), res.Error(1)
}

func (wds writeDonerSpy) Done() {
	wds.spy.Called()
}

//
// StandardVariableFields are the fields that are expected to be in (almost)
// every Sous log message, but that will be difficult to predict.
// Use this var with AssertMessageFields as a starter for the variableFields argument.
//
// For example:
//   logging.AssertMessageFields(t, msg, logging.StandardVariableFields, map[string]interface{}{...})
var StandardVariableFields = []string{
	"@timestamp",
	"call-stack-trace",
	"call-stack-file",
	"call-stack-line-number",
	"call-stack-function",
	"thread-name",
}

var unprefixedDeployerFields = []string{
	"sous-manifest-id",
	"sous-deployment-diffs",
	"sous-deployment-id",
	"sous-diff-disposition",

}

var prefixedDeployerFields = []string{
	"%s-artifact-name",
	"%s-artifact-qualities",
	"%s-artifact-type",
	"%s-checkready-failurestatuses",
	"%s-checkready-portindex",
	"%s-checkready-protocol",
	"%s-checkready-interval",
	"%s-checkready-retries",
	"%s-checkready-uripath",
	"%s-checkready-uritimeout",
	"%s-clustername",
	"%s-env",
	"%s-flavor",
	"%s-kind",
	"%s-metadata",
	"%s-numinstances",
	"%s-offset",
	"%s-owners",
	"%s-repo",
	"%s-resources",
	"%s-startup-connectdelay",
	"%s-startup-connectinterval",
	"%s-startup-skipcheck",
	"%s-startup-timeout",
	"%s-status",
	"%s-tag",
	"%s-volumes",
}

func StandardDeployerFields(prefix1 string, prefix2 string) ([]string) {
	deployerFields := unprefixedDeployerFields
	deployerFields = append(deployerFields, StandardVariableFields...)
	for _, str := range prefixedDeployerFields {
		deployerFields = append(deployerFields, fmt.Sprintf(str, prefix1))
		deployerFields = append(deployerFields, fmt.Sprintf(str, prefix2))
	}
	return deployerFields
}

// IntervalVariableFields are the fields generated by Intervals when they're closed.
// Use this var with AssertMessageFields when your message includes an Interval like
//   logging.AssertMessageFields(t, msg, logging.IntervalVariableFields, map[string]interface{}{...})
// (for incomplete intervals, just add "started-at" to the variableFields.)
var IntervalVariableFields = []string{"started-at", "finished-at", "duration"}

// AssertMessageFields is a testing function - it receives an eachFielder and confirms that it:
//  * generates no duplicate fields
//  * generates fields with the names in variableFields, and ignores their values
//  * generates fields with the names and values in fixedFields
//  * generates an @loglov3-otl field
// Additionally, if the test passes, a rough proposal of an "OTL" schema definition
// will be written to a tempfile.
//
// See also the StandardVariableFields and IntervalVariableFields convenience variables.
func AssertMessageFields(t *testing.T, msg eachFielder, variableFields []string, fixedFields map[string]interface{}) {
	t.Helper()

	assert.Contains(t, fixedFields, "@loglov3-otl", "Structured log entries need @loglov3-otl or will be DLQ'd")
	rawAssertMessageFields(t, msg, variableFields, fixedFields)
}

func rawAssertMessageFields(t *testing.T, msg eachFielder, variableFields []string, fixedFields map[string]interface{}) {
	t.Helper()

	actualFields := map[string]interface{}{}

	msg.EachField(func(name string, value interface{}) {
		assert.NotContains(t, actualFields, name) //don't clobber a field
		actualFields[name] = value
		switch name {
		case "@timestamp", "started-at", "finished-at": // known timestamp fields
			if assert.IsType(t, "", value) {
				assert.Regexp(t, `.*Z$`, value.(string), "%s: %q not in UTC!", name, value)
			}
		}
	})

	for _, f := range variableFields {
		assert.Contains(t, actualFields, f)
		delete(actualFields, f)
	}

	assert.Equal(t, fixedFields, actualFields)

	// If the test passes, write a proposed OTL to a tempfile and report the path.
	// These are super useful for updating the logging schemas,
	// and get us a long way toward aligning our fields with theirs.
	if _, hasOTL := actualFields["@loglov3-otl"]; !t.Failed() && hasOTL {
		tmpfile, err := ioutil.TempFile("", actualFields["@loglov3-otl"].(string))
		if err != nil {
			t.Logf("Problem trying to open file to write proposed OTL: %v", err)
			return
		}
		otl := map[string]interface{}{
			"otl": map[string]interface{}{
				"name":        actualFields["@loglov3-otl"],
				"description": "<description goes here>",
				"owners":      []string{"sous-team"},
				"inherits":    []string{"ot-v1", "call-stack-v1"},
			},
			"fields": map[string]interface{}{},
		}

		msg.EachField(func(n string, v interface{}) {
			switch n {
			case "call-stack-line-number", "call-stack-function", "call-stack-file", "@timestamp", "thread-name", "@loglov3-otl":
				return
			}
			switch v.(type) {
			default:
				t.Errorf("Don't know the OTL type for %v %[1]t", v)
				return
			case string:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "string", "optional": true}
			case bool:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "boolean", "optional": true}
			case int32, uint32:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "int", "optional": true}
			case int, uint, int64, uint64:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "long", "optional": true}
			case float32, float64:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "float", "optional": true}
			case time.Time:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "timestamp", "optional": true}
			}
		})

		b, err := yaml.Marshal(otl)
		if err != nil {
			t.Logf("Problem trying to serialize proposed OTL: %v", err)
			return
		}
		if _, err = tmpfile.Write(b); err == nil {
			t.Logf("Proposed OTL written to %q", tmpfile.Name())
		} else {
			t.Logf("Problem trying to write proposed OTL: %v", err)
		}
	}
}

// AssertConfiguration is a testing method - it allows us to test that certain configuration values are as expected.
func AssertConfiguration(ls *LogSet, graphiteURL string) error {
	addr, err := net.ResolveTCPAddr("tcp", graphiteURL)
	if err != nil {
		return err
	}
	if ls.dumpBundle == nil {
		return fmt.Errorf("dumpBundle is nil!")
	}
	if ls.dumpBundle.graphiteConfig == nil {
		return fmt.Errorf("graphiteConfig is nil!")
	}
	if ls.dumpBundle.graphiteConfig.Addr.String() != addr.String() {
		return fmt.Errorf("Graphite URL was %q not %q(%s)", ls.dumpBundle.graphiteConfig.Addr, addr, graphiteURL)
	}
	return nil
}
