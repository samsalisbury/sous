package logging

import (
	"fmt"
	"io"
	"os"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

type (
	// LogSet is the stopgap for a decent injectable logger
	LogSet struct {
		Debug  logwrapper
		Info   logwrapper
		Warn   logwrapper
		Notice logwrapper
		Vomit  logwrapper

		level uint

		name string

		metrics metrics.Registry

		err io.Writer

		logrus *logrus.Logger

		liveConfig *Config

		console io.Writer
	}

	// A temporary type until we can stop using the LogSet loggers directly
	// XXX remove and fix accesses to Debug, Info, etc. to be Debugf etc
	logwrapper func(string, ...interface{})
)

var (
	// Log collects various loggers to use for different levels of logging
	// XXX A goal should be to remove this global, and instead inject logging where we need it.
	//
	// Notice that the global LotSet doesn't have metrics available - when you
	// want metrics in a component, you need to add an injected LogSet. c.f.
	// ext/docker/image_mapping.go
	Log = func() LogSet {
		return *(NewLogSet("", os.Stderr))
	}()
)

func (w logwrapper) Printf(f string, vs ...interface{}) {
	w(f, vs...)
}

func (w logwrapper) Print(vs ...interface{}) {
	w(fmt.Sprint(vs...))
}

func (w logwrapper) Println(vs ...interface{}) {
	w(fmt.Sprint(vs...))
}

// SilentLogSet returns a logset that discards everything by default
func SilentLogSet() *LogSet {
	ls := NewLogSet("", os.Stderr)
	ls.BeQuiet()
	return ls
}

// NewLogSet builds a new Logset that feeds to the listed writers
// If name is "", no metric collector will be built, and all metrics provided
// by this logset will be bitbuckets.
func NewLogSet(name string, err io.Writer) *LogSet {
	// logrus uses a pool for entries, which means we probably really should only have one.
	// this means that output configuration and level limiting is global to the logset and
	// its children.
	lgrs := logrus.New()
	lgrs.Out = err
	ls := newls(name, err, lgrs)
	ls.imposeLevel()
	if name != "" {
		ls.metrics = metrics.NewPrefixedRegistry(name + ".")
	}
	return ls
}

// Child produces a child logset, namespaced under "name".
func (ls *LogSet) Child(name string) *LogSet {
	child := newls(ls.name+"."+name, ls.err, ls.logrus)
	child.level = ls.level
	child.imposeLevel()
	if ls.metrics != nil {
		child.metrics = metrics.NewPrefixedChildRegistry(ls.metrics, name+".")
	}
	return child
}

func newls(name string, err io.Writer, lgrs *logrus.Logger) *LogSet {
	ls := &LogSet{
		err:    err,
		name:   name,
		level:  1,
		logrus: lgrs,
	}
	ls.Warn = logwrapper(func(f string, as ...interface{}) { ls.warnf(f, as) })
	ls.Notice = ls.Warn
	ls.Info = ls.Warn
	ls.Debug = logwrapper(func(f string, as ...interface{}) { ls.debugf(f, as) })
	ls.Vomit = logwrapper(func(f string, as ...interface{}) { ls.vomitf(f, as) })
	ls.console = os.Stderr

	return ls
}

// Configure allows an existing LogSet to change its settings.
func (ls *LogSet) Configure(cfg Config) error {
	err := ls.configureKafka(cfg)
	if err != nil {
		return err
	}

	ls.liveConfig = cfg
}

func (ls LogSet) configureKafka(cfg Config) error {
	if ls.liveConfig != nil && ls.liveConfig.useKafka() {
		if cfg.useKafka() {
			reportLogConfigurationWarning(ls, "cannot reconfigure kafka")
		} else {
			reportLogConfigurationWarning(ls, "cannot disable kafka")
		}
		return
	}

	if !cfg.useKafka() {
		return
	}

	hook, err := kafkalogrus.NewKafkaLogrusHook("kafkahook",
		cfg.getKafkaLevels(),
		&logrus.JSONFormatter{},
		cfg.getBrokers(),
		cfg.Kafka.Topic,
		false)

	// One cause of errors: can't reach any brokers
	// c.f. https://github.com/Shopify/sarama/blob/master/client.go#L114
	if err != nil {
		return err
	}

	ls.logrus.AddHook(hook)
}

// Console implements LogSink on LogSet
func (ls LogSet) Console() io.Writer {
	return ls.console
}

// Vomitf is a simple wrapper on Vomit.Printf
func (ls LogSet) Vomitf(f string, as ...interface{}) { ls.vomitf(f, as...) }
func (ls LogSet) vomitf(f string, as ...interface{}) {
	m := NewGenericMsg(ExtraDebugLevel1, fmt.Sprintf(f, as...), nil)
	Deliver(m, ls)
}

// Debugf is a simple wrapper on Debug.Printf
func (ls LogSet) Debugf(f string, as ...interface{}) { ls.debugf(f, as...) }
func (ls LogSet) debugf(f string, as ...interface{}) {
	m := NewGenericMsg(DebugLevel, fmt.Sprintf(f, as...), nil)
	Deliver(m, ls)
}

func (ls LogSet) Warnf(f string, as ...interface{}) { ls.warnf(f, as...) }
func (ls LogSet) warnf(f string, as ...interface{}) {
	m := NewGenericMsg(WarningLevel, fmt.Sprintf(f, as...), nil)
	Deliver(m, ls)
}

func (ls LogSet) imposeLevel() {
	ls.logrus.SetLevel(logrus.ErrorLevel)

	if ls.level >= 1 {
		ls.logrus.SetLevel(logrus.WarnLevel)
	}

	if ls.level >= 2 {
		ls.logrus.SetLevel(logrus.DebugLevel)
	}

	if ls.level >= 3 {
		ls.logrus.SetLevel(logrus.DebugLevel)
	}
}

// BeQuiet gets the LogSet to discard all its output
func (ls LogSet) BeQuiet() {
	ls.level = 0
	ls.imposeLevel()
}

// BeTerse gets the LogSet to print debugging output
func (ls LogSet) BeTerse() {
	ls.level = 1
	ls.imposeLevel()
}

// BeHelpful gets the LogSet to print debugging output
func (ls LogSet) BeHelpful() {
	ls.level = 2
	ls.imposeLevel()
}

// BeChatty gets the LogSet to print all its output - useful for temporary debugging
func (ls LogSet) BeChatty() {
	ls.level = 3
	ls.imposeLevel()
}
