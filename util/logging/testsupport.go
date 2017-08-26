package logging

import "github.com/nyarly/spies"

type logSinkSpy struct {
	*spies.Spy
}

func newLogSinkSpy() (logSink, *spies.Spy) {
	spy := spies.NewSpy()
	return logSinkSpy{spy}, spy
}
