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
		Debug  *log.Logger
		Info   *log.Logger
		Warn   *log.Logger
		Notice *log.Logger
		Vomit  *log.Logger
	}
)

var (
	// Log collects various loggers to use for different levels of logging
	Log = func() LogSet {
		return *(NewLogSet(os.Stderr, ioutil.Discard, ioutil.Discard))
	}()
)

// SilentLogSet returns a logset that discards everything by default
func SilentLogSet() *LogSet {
	return NewLogSet(ioutil.Discard, ioutil.Discard, ioutil.Discard)
}

// NewLogSet builds a new Logset that feeds to the listed writers
func NewLogSet(warn, debug, vomit io.Writer) *LogSet {
	warnLogger := log.New(warn, "warn: ", 0)
	return &LogSet{
		// Debug is a logger - use log.SetOutput to get output from
		Vomit:  log.New(vomit, "vomit: ", log.Lshortfile|log.Ldate|log.Ltime),
		Debug:  log.New(debug, "debug: ", log.Lshortfile|log.Ldate|log.Ltime),
		Info:   warnLogger, // XXX deprecate Info
		Notice: warnLogger, // XXX deprecate Notice
		Warn:   warnLogger,
	}
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
