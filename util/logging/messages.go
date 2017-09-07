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

import (
	"runtime"
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
	LogSink interface {
		LogMessage(Level, LogMessage)

		ClearCounter(name string)
		IncCounter(name string, amount int64)
		DecCounter(name string, amount int64)

		UpdateTimer(name string, dur time.Duration)
		UpdateTimerSince(name string, time time.Time)

		UpdateSample(name string, value int64)

		Console() io.Writer
	}

	LogMessage interface {
		DefaultLevel() Level
		Message() string
		EachField(FieldReportFn)
	}

	MetricsMessage interface {
		MetricsTo(LogSink)
	}

	ConsoleMessage interface {
		WriteToConsole(console io.Writer)
	}

	// CallTime captures the time at which a log message was generated.
	CallTime time.Time

	FieldReportFn func(string, interface{})

	// Level is the "level" of a log message (e.g. debug vs fatal)
	Level int
	// error interface{}

)

const (
	// CriticalLevel is the level for logging critical errors.
	CriticalLevel = Level(iota)

	// WarningLevel is the level for messages that may be problematic.
	WarningLevel = Level(iota)

	// InformationLevel is for messages generated during normal operation.
	InformationLevel = Level(iota)

	// DebugLevel is for messages primarily of interest to the software's developers.
	DebugLevel = Level(iota)

	// ExtraDebugLevel1 is the first level of "super" debug messages.
	ExtraDebugLevel1 = Level(iota)
)

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

func Deliver(message interface{}, logger LogSink) {
	if lm, is := message.(LogMessage); is {
		Level := getLevel(lm)
		logger.LogMessage(Level, lm)
	}

	if mm, is := message.(MetricsMessage); is {
		mm.MetricsTo(logger)
	}

	if cm, is := message.(ConsoleMessage); is {
		cm.WriteToConsole(logger.Console())
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

// GetCallTime captures the current call time.
func GetCallTime() CallTime {
	return CallTime(time.Now())
}

func (lvl Level) DefaultLevel() Level {
	return lvl
}

func (time CallTime) EachField(f FieldReportFn) {
	f("time", time)
}
