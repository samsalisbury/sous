package logging

import (
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

	counterSpy struct {
		spy *spies.Spy
	}
	counterController struct {
		*spies.Spy
	}

	timerSpy struct {
		spy *spies.Spy
	}
	timerController struct {
		*spies.Spy
	}

	updaterSpy struct {
		spy *spies.Spy
	}
	updaterController struct {
		*spies.Spy
	}

	metricsControllers struct {
		counter counterController
		timer   timerController
		updater updaterController
	}
)

func newLogSinkSpy() (logSink, logSinkController) {
	spy := spies.NewSpy()
	return logSinkSpy{spy: spy}, logSinkController{spy}
}

func newCounterSpy() (Counter, counterController) {
	spy := spies.NewSpy()
	return counterSpy{spy: spy}, counterController{spy}
}

func newTimerSpy() (Timer, timerController) {
	spy := spies.NewSpy()
	return timerSpy{spy: spy}, timerController{spy}
}

func newUpdaterSpy() (Updater, updaterController) {
	spy := spies.NewSpy()
	return updaterSpy{spy: spy}, updaterController{spy}
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
	res := lss.spy.Called(name)
	return res.Get(0).(Counter)
}

func (lss logSinkSpy) GetTimer(name string) Timer {
	res := lss.spy.Called(name)
	return res.Get(0).(Timer)
}

func (lss logSinkSpy) GetUpdater(name string) Updater {
	res := lss.spy.Called(name)
	return res.Get(0).(Updater)
}

func (lss logSinkSpy) LogMessage(l level, m logMessage) {
	lss.spy.Called(l, m)
}

func (lsc logSinkController) setupDefaultMetrics() metricsControllers {
	cs, cc := newCounterSpy()
	ts, tc := newTimerSpy()
	us, uc := newUpdaterSpy()

	lsc.MatchMethod("GetCounter", spies.AnyArgs, cs)
	lsc.MatchMethod("GetTimer", spies.AnyArgs, ts)
	lsc.MatchMethod("GetUpdater", spies.AnyArgs, us)

	return metricsControllers{cc, tc, uc}
}

func (cs counterSpy) Clear() {
	cs.spy.Called()
}

func (cs counterSpy) Inc(i int64) {
	cs.spy.Called(i)
}

func (cs counterSpy) Dec(i int64) {
	cs.spy.Called(i)
}

func (ts timerSpy) Time(f func()) {
	ts.spy.Called(f)
	f()
}

func (ts timerSpy) Update(d time.Duration) {
	ts.spy.Called(d)
}

func (ts timerSpy) UpdateSince(t time.Time) {
	ts.spy.Called(t)
}

func (us updaterSpy) Update(i int64) {
	us.spy.Called(i)
}
