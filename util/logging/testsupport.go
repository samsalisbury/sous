package logging

import "github.com/nyarly/spies"

type logSinkSpy struct {
	*spies.Spy
}

func newLogSinkSpy() (logSink, *spies.Spy) {
	spy := spies.NewSpy()
	return logSinkSpy{spy}, spy
}

/*
	messageSink interface {
		LogMessage(level, logMessage)
	}

	metricsSink interface {
		GetTimer(name string) Timer
		GetCounter(name string) Counter
		GetUpdater(name string) Updater
	}
*/

func (lss logSinkSpy) GetCounter(name string) Counter {
	res := lss.Spy.Called(name)
	return res.Get(0).(Counter)
}

func (lss logSinkSpy) GetTimer(name string) Timer {
	res := lss.Spy.Called(name)
	return res.Get(0).(Timer)
}

func (lss logSinkSpy) GetUpdater(name string) Updater {
	res := lss.Spy.Called(name)
	return res.Get(0).(Updater)
}

func (lss logSinkSpy) LogMessage(l level, m logMessage) {
	lss.Spy.Called(l, m)
}
