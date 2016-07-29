package sous

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type (
	// ILogger is like this:
	// XXX This is a complete placeholder for work in the ilog branch
	// I needed some extra logging for config process, and didn't want to double
	// down on a process we knew we were going to abandon
	ILogger interface {
		SetLogFunc(func(...interface{}))
		SetDebugFunc(func(...interface{}))
	}
)

var (
	// Log collects various loggers to use for different levels of logging
	Log = struct {
		Debug  *log.Logger
		Info   *log.Logger
		Warn   *log.Logger
		Notice *log.Logger
		Vomit  *log.Logger
	}{
		// Debug is a logger - use log.SetOutput to get output from
		Vomit:  log.New(ioutil.Discard, "vomit: ", log.Lshortfile),
		Debug:  log.New(ioutil.Discard, "debug: ", log.Lshortfile),
		Info:   log.New(ioutil.Discard, "info: ", 0),
		Notice: log.New(ioutil.Discard, "notice: ", 0),
		Warn:   log.New(os.Stderr, "warn: ", 0),
	}
)

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
