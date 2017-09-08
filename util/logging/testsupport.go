package logging

import (
	"io"
	"time"

	"github.com/nyarly/spies"
)

type (
	logSinkSpy struct {
		spy *spies.Spy
	}

	logSinkController struct {
		*spies.Spy
	}
)

func NewLogSinkSpy() (LogSink, logSinkController) {
	spy := spies.NewSpy()
	return logSinkSpy{spy: spy}, logSinkController{spy}
}

func (lss logSinkSpy) LogMessage(lvl Level, msg LogMessage) {
	lss.spy.Called(lvl, msg)
}

func (lss logSinkSpy) ClearCounter(name string) {
	lss.spy.Called(name)
}

func (lss logSinkSpy) IncCounter(name string, amount int64) {
	lss.spy.Called(name, amount)
}

func (lss logSinkSpy) DecCounter(name string, amount int64) {
	lss.spy.Called(name, amount)
}

func (lss logSinkSpy) UpdateTimer(name string, dur time.Duration) {
	lss.spy.Called(name, dur)
}

func (lss logSinkSpy) UpdateTimerSince(name string, time time.Time) {
	lss.spy.Called(name, time)
}

func (lss logSinkSpy) UpdateSample(name string, value int64) {
	lss.spy.Called(name, value)
}

func (lss logSinkSpy) Console() io.Writer {
	res := lss.spy.Called()
	return res.Get(0).(io.Writer)
}
