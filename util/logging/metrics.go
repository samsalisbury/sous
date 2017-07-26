package logging

import (
	"net/http"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
)

const (
	defRezSize    = 1024
	defDecayAlpha = 0.15
)

type (
	// jdl: Not yet appearing:
	// Float64Gauges
	// EWMAs or Unix-load style Meters
	// Uniform sampling

	// Counter is a write-only interface for an integer counter
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

	multiUpdate struct {
		decSample metrics.Sample
		uniSample metrics.Sample
		last      metrics.Gauge
	}
)

func (u *multiUpdate) Update(n int64) {
	u.decSample.Update(n)
	u.uniSample.Update(n)
	u.last.Update(n)
}

// HasMetrics indicates whether this LogSet has been configured with metrics
func (ls *LogSet) HasMetrics() bool {
	return ls.metrics != nil
}

// ExpHandler returns an http.Handler to export metrics registered with this LogSet.
// panics if the LogSet hasn't been set up with metrics yet.
func (ls *LogSet) ExpHandler() http.Handler {
	if ls.metrics == nil {
		panic("LogSet metric unset!")
	}
	return exp.ExpHandler(ls.metrics)
}

// GetTimer returns a timer so that components can record timing metrics.
func (ls *LogSet) GetTimer(name string) Timer {
	if ls.metrics == nil {
		return metrics.NilTimer{}
	}
	return metrics.GetOrRegisterTimer(name, ls.metrics)
}

// GetCounter returns a counter so that components can count things.
func (ls *LogSet) GetCounter(name string) Counter {
	if ls.metrics == nil {
		return metrics.NilCounter{}
	}
	return metrics.GetOrRegisterCounter(name, ls.metrics)
}

// GetUpdater returns an updater that records both the immediate value and a decaying sample.
func (ls *LogSet) GetUpdater(name string) Updater {
	if ls.metrics == nil {
		return metrics.NilGauge{}
	}
	ds := metrics.NewExpDecaySample(defRezSize, defDecayAlpha)
	metrics.GetOrRegisterHistogram(name+".decay", ls.metrics, ds)

	us := metrics.NewUniformSample(defRezSize)
	metrics.GetOrRegisterHistogram(name+".uniform", ls.metrics, us)

	g := metrics.GetOrRegisterGauge(name+".last", ls.metrics)

	return &multiUpdate{
		decSample: ds,
		uniSample: us,
		last:      g,
	}
}
