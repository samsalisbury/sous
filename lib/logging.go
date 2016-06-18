package sous

import (
	"io/ioutil"
	"log"
	"os"
)

var (
	// Log collects various loggers to use for different levels of logging
	Log = struct {
		Debug *log.Logger
		Info  *log.Logger
		Warn  *log.Logger
	}{
		// Debug is a logger - use log.SetOutput to get output from
		Debug: log.New(ioutil.Discard, "debug: ", log.Lshortfile),
		Info:  log.New(ioutil.Discard, "info: ", 0),
		Warn:  log.New(os.Stderr, "warn: ", 0),
	}
)
