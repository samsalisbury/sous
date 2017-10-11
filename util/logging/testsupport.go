package logging

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/nyarly/spies"
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

	logSinkController struct {
		*spies.Spy
		Metrics metricsSinkController
		Console writeDonerController
	}
)

// NewLogSinkSpy returns a spy/controller pair
func NewLogSinkSpy() (LogSink, logSinkController) {
	spy := spies.NewSpy()

	console, cc := NewWriteDonerSpy()
	metrics, mc := NewMetricsSpy()

	ctrl := logSinkController{
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
	m := NewGenericMsg(ExtraDebug1Level, fmt.Sprintf(f, as...), nil)
	Deliver(m, lss)
}

func (lss logSinkSpy) Debugf(f string, as ...interface{}) {
	m := NewGenericMsg(DebugLevel, fmt.Sprintf(f, as...), nil)
	Deliver(m, lss)
}

func (lss logSinkSpy) Warnf(f string, as ...interface{}) {
	m := NewGenericMsg(WarningLevel, fmt.Sprintf(f, as...), nil)
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

// AssertMessageFields is a testing function - it receives an eachFielder and confirms that it:
//  * generates no duplicate fields
//  * generates fields with the names in variableFields, and ignores their values
//  * generates fields with the names and values in fixedFields
//  * generates an @loglov3-otl field
func AssertMessageFields(t *testing.T, msg eachFielder, variableFields []string, fixedFields map[string]interface{}) {
	actualFields := map[string]interface{}{}

	msg.EachField(func(name string, value interface{}) {
		assert.NotContains(t, actualFields, name) //don't clobber a field
		actualFields[name] = value
	})

	assert.Contains(t, actualFields, "@loglov3-otl") // if this is missing, we DLQ
	for _, f := range variableFields {
		assert.Contains(t, actualFields, f)
		delete(actualFields, f)
	}

	assert.Equal(t, fixedFields, actualFields)

	if !t.Failed() {
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

		for n, v := range actualFields {
			switch val := v.(type) {
			default:
				t.Logf("Don't know the OTL type for %v %[1]t", v)
				return
			case string:
				if val == "@loglov3-otl" {
					continue
				}
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "string", "optional": true}
			case bool:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "bool", "optional": true}
			case int32, uint32:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "int", "optional": true}
			case int, uint, int64, uint64:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "long", "optional": true}
			case float32, float64:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "float", "optional": true}
			case time.Time:
				otl["fields"].(map[string]interface{})[n] = map[string]interface{}{"type": "timestamp", "optional": true}
			}
		}

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
