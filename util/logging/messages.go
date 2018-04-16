// The goal of this package is to integrate structured logging an metrics
// reporting with error handling in an interface as close as possible to the
// fluency of fmt.Errorf(...)

// or of errors.Wrapf(err, "fmt", args...)

// Concerns:
//   a. structured logging using a defined scheme
//   b. build-time checking of errors
//   c. 3 purposes, which each message type can make use of 1-3 of:
//     logging to ELK,
//     metrics collection
//     error reporting
//   d. Contextualization - i.e. pull message fields from a context.Context
//        or from a logging context likewise contextualized.
//   e. ELK specific fields (i.e. "this is schema xyz")

// Nice to have:
//   z. Output filtering disjoint from creation (i.e. *not* log.debug but rather debug stuff from the singularity API)
//   y. Runtime output filtering, via e.g. HTTP requests.
//   x. A live ringbuffer of all messages

// b & d are in tension.
// also, a with OTLs, because optional fields

package logging

// go get github.com/opentable/go-loglov3-gen (private repo)
//go:generate go-loglov3-gen -loglov3-dir $LOGLOV3_DIR -output-dir .

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type (
	messageSink interface {
	}

	/*
		Counter interface {
			Clear()
			Inc(int64)
			Dec(int64)
		}

		// Timer is a write-only interface over a timer.
		Timer interface {
			Time(func())
			Update(time.Duration)
			UpdateSince(time.Time)
		}

		// Updater is a generalization of write-only metrics - integers that can be set.
		// e.g. simple gauges or analyzed samples etc.
		Updater interface {
			Update(int64)
		}

	*/

	// A LogSink can be used in Deliver to send messages for logging.
	LogSink interface {
		// Child returns a namespaced child, with a set of EachFielders for context.
		Child(name string, context ...EachFielder) LogSink

		// Fields is used to record the name/value fields of a structured message.
		Fields([]EachFielder)

		// Metrics returns a MetricsSink, which will be used to record MetricsMessages.
		Metrics() MetricsSink

		// Console returns a WriteDoner, which will be used to record ConsoleMessages.
		Console() WriteDoner

		// ExtraConsole returns a WriteDoner, which will be used to record ExtraConsoleMessages.
		ExtraConsole() WriteDoner

		// AtExit() does last-minute cleanup of stuff
		AtExit()

		// ForceDefer is used during testing to suspend the "panic during testing" behavior.
		ForceDefer() bool
	}

	// A MetricsSink is passed into a MetricsMessage's MetricsTo(), so that the
	// it can record its metrics. Once done, the Done method is called - if the
	// metrics are incomplete or insistent, the MetricsSink can then report
	// errors.
	// xxx this facility is preliminary, and Sous doesn't yet record these errors.
	MetricsSink interface {
		ClearCounter(name string)
		IncCounter(name string, amount int64)
		DecCounter(name string, amount int64)

		UpdateTimer(name string, dur time.Duration)
		UpdateTimerSince(name string, time time.Time)

		UpdateSample(name string, value int64)

		Done()
	}

	// WriteDoner is like a WriteCloser, but the Done message also asserts that something useful was written
	// After a console message has been written, the Done method is called, so
	// that the WriteDoner can report about badly formed or missing console
	// messages.
	// xxx this facility is preliminary, and Sous doesn't yet record these errors.
	WriteDoner interface {
		io.Writer
		Done()
	}

	writeDoner struct {
		io.Writer
	}

	// A EachFielder provides EachField - which calls its argument for each field it wants to submit for logging.
	EachFielder interface {
		EachField(fn FieldReportFn)
	}

	// A LevelRecommender can recommend a log level.
	LevelRecommender interface {
		RecommendedLevel() Level
	}

	// Submessage is not a complete message on its own
	Submessage interface {
		EachFielder
		LevelRecommender
	}

	// OldLogMessage captures a deprecated interface
	// prefer instead to use EachFielder and include Severity and Message fields.
	// Don't do both though; make a clean break with this interface.
	OldLogMessage interface {
		// The severity level of this message, potentially (in the future) manipulated
		// by dynamic rules.
		DefaultLevel() Level
		// A simple textual message describing the logged event. Usually hardcoded (or almost so.)
		Message() string
	}

	// A LogMessage has structured data to report to the structured log server (c.f. Deliver).
	// Almost every implementation of LogMessage should include a CallerInfo.
	LogMessage interface {
		OldLogMessage
		// Called to report the individual fields for this message.
		EachField(fn FieldReportFn)
	}

	// A MetricsMessage has metrics data to record (c.f. Deliver)
	MetricsMessage interface {
		MetricsTo(MetricsSink)
	}

	// A ConsoleMessage has messages to report to a local human operator (c.f. Deliver)
	ConsoleMessage interface {
		WriteToConsole(console io.Writer)
	}

	// FieldReportFn is used by LogMessages to report their fields.
	FieldReportFn func(FieldName, interface{})

	// A MessageField is a quick wrapper for string with EachField.
	MessageField string

	kv struct {
		k FieldName
		v interface{}
	}

	// ToConsole allows quick creation of Console messages.
	ToConsole struct {
		msg interface{}
	}

	consoleMessage struct {
		ToConsole
	}
)

// All calls EachField on each of the arguments.
func (frf FieldReportFn) All(efs ...EachFielder) {
	for _, ef := range efs {
		ef.EachField(frf)
	}
}

// KV creates a single-entry EachFielder with the FieldName as the name.
func KV(n FieldName, v interface{}) EachFielder {
	return kv{k: n, v: v}
}

func (p kv) EachField(fn FieldReportFn) {
	fn(p.k, p.v)
}

// EachField implements EachFielder on MessageField.
func (m MessageField) EachField(fn FieldReportFn) {
	fn(CallStackMessage, string(m))
}

func (m MessageField) String() string {
	return string(m)
}

// WriteToConsole implements ConsoleMessage on ToConsole.
func (tc ToConsole) WriteToConsole(c io.Writer) {
	fmt.Fprintf(c, fmt.Sprintf("%s\n", tc.msg))
}

// Console marks a string as being suitable for console output.
func Console(m interface{}) ToConsole {
	return ToConsole{msg: m}
}

// ConsoleAndMessage wraps a string such that it will be both a console output and the primary message of a log entry.
func ConsoleAndMessage(m interface{}) consoleMessage {
	return consoleMessage{ToConsole{msg: m}}
}

func (m consoleMessage) EachField(fn FieldReportFn) {
	fn(CallStackMessage, fmt.Sprintf("%s", m.msg))
}

/*
  A static analysis approach here would:

	Check that the JSON tags on structs matched the schemas they claim.
	Check that schema-required fields tie with params to the contructor.
	Maybe check that contexted messages were always receiving contexts with the right WithValues

	A code generation approach would:

	Take the schemas and produce structs with JSON tags
	Produce constructors for the structs with the required fields.
	Produce LogXXX methods and functions around those constructors.

	We can live without those, probably, if we build the interfaces *as if*...

*/

func nopDoner(w io.Writer) WriteDoner {
	return &writeDoner{w}
}

func (writeDoner) Done() {}

// Deliver is the core of the logging messages design.
//
// The message argument may implement
// any of LogMessage, MetricsMessage or ConsoleMessage, and the
// data contained in the message will be dispatched appropriately.
//
// Furthermore, messages that don't implement any of those interfaces,
// or which panic when operated upon,
// themselves generate a well-tested message so that they can be caught and fixed.
//
// The upshot is that messages can be Delivered on the spot and
// later determine what facilities are appropriate.
func Deliver(logger LogSink, messages ...interface{}) {
	if logger == nil {
		panic("null logger")
	}

	//determine if function running under test, allow overwritten value from options functions
	testFlag := strings.HasSuffix(os.Args[0], ".test")
	if logger.ForceDefer() {
		testFlag = false
	}
	if !testFlag {
		defer loggingPanicsShouldntCrashTheApp(logger, messages)
	}

	items := partitionItems(messages)

	logger.Fields(items.eachFielders)

	metrics := logger.Metrics()
	for _, mm := range items.metricsMessages {
		mm.MetricsTo(metrics)
	}
	metrics.Done()

	for _, cm := range items.consoleMessages {
		cm.WriteToConsole(logger.Console())
	}

	if _, dont := messages[0].(*silentMessageError); items.silent() && !dont {
		reportSilentMessage(logger, messages)
	}
}

type partitionedItems struct {
	eachFielders    []EachFielder
	consoleMessages []ConsoleMessage
	metricsMessages []MetricsMessage
}

// holding item partitioning to logging time.
func partitionItems(items []interface{}) partitionedItems {
	l := partitionedItems{}
	others := []interface{}{}

	for _, item := range items {
		ef, isef := item.(EachFielder)
		olm, isolm := item.(OldLogMessage)
		cm, iscm := item.(ConsoleMessage)
		mm, ismm := item.(MetricsMessage)
		if isef {
			if isolm {
				m := olm.Message()
				lvl := olm.DefaultLevel()
				l.eachFielders = append(l.eachFielders, MessageField(m), lvl)
			}
			l.eachFielders = append(l.eachFielders, ef)
		}
		if iscm {
			l.consoleMessages = append(l.consoleMessages, cm)
		}
		if ismm {
			l.metricsMessages = append(l.metricsMessages, mm)
		}
		if !(isef || iscm || ismm) {
			others = append(others, item)
		}
	}
	if !l.silent() && len(others) > 0 {
		l.eachFielders = append(l.eachFielders, assembleStrayFields(others...))
	}
	return l
}

func (i partitionedItems) silent() bool {
	if len(i.eachFielders) > 0 {
		return false
	}
	if len(i.consoleMessages) > 0 {
		return false
	}
	if len(i.metricsMessages) > 0 {
		return false
	}
	return true
}

// a fake "message" designed to trigger the well-tested silentMessageError
type loggingPanicFakeMessage struct {
	broken interface{}
}

// granted that logging can be set up in the first place,
// problems with a logging message should not crash the whole app
// therefore: recover the panic do the simplest thing that will be logged,
func loggingPanicsShouldntCrashTheApp(ls LogSink, msg interface{}) {
	if rec := recover(); rec != nil {
		Deliver(ls, loggingPanicFakeMessage{msg})
	}
}

// ClearCounter implements part of LogSink on LogSet
func (ls LogSet) ClearCounter(name string) {
	ls.GetCounter(name).Clear()
}

// IncCounter implements part of LogSink on LogSet
func (ls LogSet) IncCounter(name string, amount int64) {
	ls.GetCounter(name).Inc(amount)
}

// DecCounter implements part of LogSink on LogSet
func (ls LogSet) DecCounter(name string, amount int64) {
	ls.GetCounter(name).Dec(amount)
}

// UpdateTimer implements part of LogSink on LogSet
func (ls LogSet) UpdateTimer(name string, dur time.Duration) {
	ls.GetTimer(name).Update(dur)
}

// UpdateTimerSince implements part of LogSink on LogSet
func (ls LogSet) UpdateTimerSince(name string, time time.Time) {
	ls.GetTimer(name).UpdateSince(time)
}

// UpdateSample implements part of LogSink on LogSet
func (ls LogSet) UpdateSample(name string, value int64) {
	ls.GetUpdater(name).Update(value)
}

// The plan here is to be able to extend this behavior such that e.g. the rules
// for levels of messages can be configured or updated at runtime.
func getLevel(lm LogMessage) Level {
	return lm.DefaultLevel()
}

// ConsoleError receives a ConsoleMessage and returns the string as it would be printed to the console.
// This can be used to implement the error interface on ConsoleMessages
func ConsoleError(msg ConsoleMessage) string {
	buf := &bytes.Buffer{}
	msg.WriteToConsole(buf)
	return buf.String()
}

// DefaultLevel is a convenience - by embedding a Level, a message can partially implement LogMessage
func (lvl Level) DefaultLevel() Level {
	return lvl
}

// EachField implements EachFielder on OTLName.
func (n OTLName) EachField(f FieldReportFn) {
	f(Loglov3Otl, n)
}
