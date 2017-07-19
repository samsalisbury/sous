package sous

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type (
	// ILogger is like this:
	// XXX This is a complete placeholder for work in the ilog branch
	// I needed some extra logging for config process, and didn't want to double
	// down on a process we knew we were going to abandon
	// XXX Further thought: I really think we should look log15 (or something) as our logging platform.
	// It won't be perfect, but it also won't suck up work
	ILogger interface {
		SetLogFunc(func(...interface{}))
		SetDebugFunc(func(...interface{}))
	}

	// LogSet is the stopgap for a decent injectable logger
	LogSet struct {
		Debug  *logwrapper
		Info   *logwrapper
		Warn   *logwrapper
		Notice *logwrapper
		Vomit  *logwrapper

		debug *log.Logger
		info  *log.Logger
		warn  *log.Logger
	}

	// A temporary type until we can stop using the LogSet loggers directly
	logwrapper struct {
		ffn func(string, ...interface{})
	}
)

var (
	// Log collects various loggers to use for different levels of logging
	// XXX A goal should be to remove this global, and instead inject logging where we need it.
	Log = func() LogSet {
		return *(NewLogSet(os.Stderr, ioutil.Discard, ioutil.Discard))
	}()
)

func (w *logwrapper) Printf(f string, vs ...interface{}) {
	w.ffn(f, vs...)
}

func (w *logwrapper) Print(f string, vs ...interface{}) {
	w.ffn(f, vs...)
}

func (w *logwrapper) Printf(f string, vs ...interface{}) {
	w.ffn(f, vs...)
}

// SilentLogSet returns a logset that discards everything by default
func SilentLogSet() *LogSet {
	return NewLogSet(ioutil.Discard, ioutil.Discard, ioutil.Discard)
}

// NewLogSet builds a new Logset that feeds to the listed writers
func NewLogSet(warn, debug, vomit io.Writer) *LogSet {
	return &LogSet{
		// Debug is a logger - use log.SetOutput to get output from
		vomit: log.New(vomit, "vomit: ", log.Lshortfile|log.Ldate|log.Ltime),
		debug: log.New(debug, "debug: ", log.Lshortfile|log.Ldate|log.Ltime),
		warn:  log.New(warn, "warn: ", 0),
	}
}

// Vomitf is a simple wrapper on Vomit.Printf
func (ls LogSet) Vomitf(f string, as ...interface{}) { ls.vomit.Printf(f, as...) }

// Debugf is a simple wrapper on Debug.Printf
func (ls LogSet) Debugf(f string, as ...interface{}) { ls.debug.Printf(f, as...) }

// Warnf is a simple wrapper on Warn.Printf
func (ls LogSet) Warnf(f string, as ...interface{}) { ls.warn.Printf(f, as...) }

// BeChatty gets the LogSet to print all its output - useful for temporary debugging
func (ls LogSet) BeChatty() {
	ls.warn.SetOutput(os.Stderr)
	ls.warn.SetFlags(log.Llongfile | log.Ltime)
	ls.vomit.SetOutput(os.Stderr)
	ls.vomit.SetFlags(log.Llongfile | log.Ltime)
	ls.debug.SetOutput(os.Stderr)
	ls.debug.SetFlags(log.Llongfile | log.Ltime)
}

// BeQuiet gets the LogSet to discard all its output
func (ls LogSet) BeQuiet() {
	ls.vomit.SetOutput(ioutil.Discard)
	ls.debug.SetOutput(ioutil.Discard)
	ls.warn.SetOutput(ioutil.Discard)
}

// SetupLogging sets up an ILogger to log into the Sous logging regime
func SetupLogging(il ILogger) {
	il.SetLogFunc(func(args ...interface{}) {
		logMaybeMap(Log.Warn, args...)
	})
	il.SetDebugFunc(func(args ...interface{}) {
		logMaybeMap(Log.Debug, args...)
	})
}

func logMaybeMap(l *log.Logger, args ...interface{}) {
	msg, mok := args[0].(string)
	fields, fok := args[1].(map[string]interface{})
	if !(mok && fok) {
		l.Println(args)
		return
	}
	msg = msg + ": "
	for k, v := range fields {
		msg = fmt.Sprintf("%s %s = %v", msg, k, v)
	}
	l.Print(msg)
	return
}
