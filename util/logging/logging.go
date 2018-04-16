// Package logging provides a framework for working with typed, structured
// log systems. Specifically, we target an ELK stack that enforces a
// whole-Elasticsearch schema for log entries. The target system drops
// messages if they do not conform to the configured schema.
//
// The design of this package is premised on using the Go type system
// to tighten the iteration cycle around working with our ELK stack.
// Specifically, we define explicit log message structs,
// and use constructor functions in order to control exactly the
// fields that have to be collected. Ideally, on-the-spot log messages
// will be guided by their constructors to provide the correct fields,
// so that we're shielded from the dead-letter queue at the time of
// logging.
//
// As a compliment to this process, we also allow for log messages
// to report metrics, and to output console messages for a human
// operator.
//
// By implementing any combination of the interfaces LogMessage,
// ConsoleMessage and MetricsMessage single log message may emit:
// structured logs to the ELK stack (LogMessage)
// textual logs to stderr (ConsoleMessage), or
// metrics messages to graphite carbon endpoints (MetricsMessage)
// Each message must implement at least one of those interfaces.
//
// The philosphy here is that one kind of reporting output
// will often compliment another, and since these outputs are predicated
// on the implementation of interfaces (c.f. LogMessage, MetricsMessage and
// ConsoleMessage), it's easy to add new reporting to a particular
// point of instrumentation in the code.
//
// If an intended message doesn't actually implement any of
// the required interfaces (e.g. by missing out implementing DefaultLevel)
// this package will create a new "silent message" log entry
// detailing as much as possible about the non-message struct.
// The same facility is used if the message causes a panic
// while reporting - the theory is that logging should never
// panic the app (even though it's very easy for it to do so.)
// It is recommended to create alerts from you ELK stack when
// "silent message log entries are created."
//
// Entry points to understanding this package are
// the Deliver function, and the LogMessage, MetricsMessage and
// ConsoleMessage interfaces.
//
// An example log message might look like this:
//  type exampleMessage struct {
//    CallerInfo
//    Level
//    myField string
//  }
//
//  func ReportExample(field string, sink LogSink) {
//    msg := newExampleMessage(field)
//    msg.CallerInfo.ExcludeMe() // this filters out ReportExample from the logged call stack
//    Deliver(msg, sink) // this is the important part
//  }
//
//  func newExampleMessage(field string) *exampleMessage {
//    return &exampleMessage{
//      CallerInfo: GetCallerInfo(NotHere()), // this formula captures the call point, while excluding the constructor
//      Level: DebugLevel,
//      myField: field,
//    }
//  }
//
//  // func (msg *exampleMessage) DefaultLevel() Level { ... }
//  // we could define this method, or let the embedded Level handle it.
//
//  func (msg *exampleMessage) Message() {
//    return msg.myField
//  }
//
//  func (msg *exampleMessage) EachField(f FieldReportFn) {
//    f("@loglov3-otl", "example-message") // a requirement of our local ELK
//    msg.CallerInfo.EachField(f) // so that the CallerInfo can register its fields
//    f("my-field", msg.myField)
//  }
//
// Final note: this package is in a state of transition. When Sous was
// starting, we bowed to indecision and simply adopted the stdlib
// "log" package, which turned out to be insufficient to our needs
// with respect to the ELK stack, and somewhat complicated to deal with
// in terms of leveled output. We're in the process of removing our
// dependencies on the "old ways," but in the meantime the interface
// is somewhat confusing, since there's two conflicted underlying approaches.
package logging

import (
	"context"
	"io"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"runtime"
	"time"

	graphite "github.com/cyberdelia/go-metrics-graphite"
	"github.com/pborman/uuid"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/samsalisbury/semv"
	"github.com/sirupsen/logrus"
)

type (
	// LogSet is the stopgap for a decent injectable logger
	//
	LogSet struct {
		level     Level
		name      string
		appRole   string
		ctxFields []EachFielder

		metrics metrics.Registry
		*dumpBundle
	}

	// ugh - I don't know what else to call this though
	dumpBundle struct {
		appIdent        *applicationID
		context         context.Context
		err, defaultErr io.Writer
		logrus          *logrus.Logger
		liveConfig      *Config
		kafkaSink       kafkaSink
		graphiteCancel  func()
		graphiteConfig  *graphite.Config
		extraConsole    io.Writer
	}
)

//RetrieveMetaData used to help retrieve more info for logging about a func
func RetrieveMetaData(f func()) (name string, uid string) {
	if p := reflect.ValueOf(f).Pointer(); p != 0 {
		if r := runtime.FuncForPC(p); r != nil {
			name = r.Name()
		}
	}
	uid = uuid.New()
	return name, uid
}

// Log collects various loggers to use for different levels of logging
// Deprecation warning: the global logger is slated for removal.
// All consumers of this package should migrate to injecting a local logger.
//
// Notice that the global LotSet doesn't have metrics available - when you
// want metrics in a component, you need to add an injected LogSet. c.f.
// ext/docker/image_mapping.go
var Log = func() LogSet { return *(SilentLogSet().Child("GLOBAL").(*LogSet)) }()

// SilentLogSet returns a logset that discards everything by default
func SilentLogSet() *LogSet {
	ls := NewLogSet(semv.MustParse("0.0.0"), "", "", os.Stderr)
	ls.BeQuiet()
	return ls
}

// NewLogSet builds a new Logset that feeds to the listed writers
func NewLogSet(version semv.Version, name string, role string, err io.Writer) *LogSet {
	// logrus uses a pool for entries, which means we probably really should only have one.
	// this means that output configuration and level limiting is global to the logset and
	// its children.
	lgrs := logrus.New()
	lgrs.Out = err

	bundle := newdb(version, err, lgrs)

	ls := newls(name, role, CriticalLevel, bundle) //level constrains Kafka output
	ls.imposeLevel()

	// use sous.<env>.<region>.*, he said
	// sous. comes from the GraphiteConfig "Prefix" field.
	// <env>.<region> from metricsScope()
	ls.metrics = metrics.NewPrefixedRegistry(bundle.appIdent.metricsScope() + ".")
	return ls
}

// Child produces a child logset, namespaced under "name".
func (ls LogSet) Child(name string, context ...EachFielder) LogSink {
	child := newls(ls.name+"."+name, ls.appRole, ls.level, ls.dumpBundle)
	child.metrics = metrics.NewPrefixedChildRegistry(ls.metrics, name+".")
	child.ctxFields = append(context, ls.ctxFields...)
	return child
}

func getEnvHash() map[string]string {
	h := map[string]string{}
	get := func(n string) {
		if v, has := os.LookupEnv(n); has {
			h[n] = v
		}
	}
	get("OT_ENV")
	get("OT_ENV_TYPE")
	get("OT_ENV_LOCATION")
	get("TASK_ID")
	get("INSTANCE_NO")
	return h
}

func newdb(vrsn semv.Version, err io.Writer, lgrs *logrus.Logger) *dumpBundle {
	env := getEnvHash()

	return &dumpBundle{
		appIdent:     collectAppID(vrsn, env),
		context:      context.Background(),
		err:          err,
		defaultErr:   err,
		logrus:       lgrs,
		extraConsole: ioutil.Discard,
	}
}

func (db *dumpBundle) replaceKafka(sink kafkaSink) {
	var old kafkaSink
	old, db.kafkaSink = db.kafkaSink, sink
	if old != nil {
		old.closedown()
	}
}

func (db *dumpBundle) sendToKafka(lvl Level, entry *logrus.Entry) error {
	if db.kafkaSink == nil {
		return nil
	}
	return db.kafkaSink.send(lvl, entry)
}

func newls(name string, role string, level Level, bundle *dumpBundle) *LogSet {
	ls := &LogSet{
		name:       name,
		appRole:    role,
		level:      level,
		dumpBundle: bundle,
	}

	return ls
}

// Configure allows an existing LogSet to change its settings.
func (ls *LogSet) Configure(cfg Config) error {
	err := ls.configureKafka(cfg)
	if err != nil {
		return err
	}

	err = ls.configureGraphite(cfg)
	if err != nil {
		return err
	}

	ls.logrus.SetLevel(cfg.getLogrusLevel())

	if cfg.Basic.DisableConsole {
		ls.dumpBundle.err = ioutil.Discard
	} else {
		ls.dumpBundle.err = ls.dumpBundle.defaultErr
	}

	if cfg.Basic.ExtraConsole {
		ls.dumpBundle.extraConsole = ls.dumpBundle.defaultErr
	} else {
		ls.dumpBundle.extraConsole = ioutil.Discard
	}

	ls.liveConfig = &cfg
	return nil
}

// AtExit implements part of LogSink on LogSet
func (ls LogSet) AtExit() {
	if ls.dumpBundle.kafkaSink != nil {
		ls.dumpBundle.kafkaSink.closedown()
	}
}

// ForceDefer returns false to register the "normal" behavior of LogSet.
func (ls LogSet) ForceDefer() bool {
	return false
}

func logrusFormatter() logrus.Formatter {
	return &logrus.JSONFormatter{
		DisableTimestamp: true, //we capture the timestamp when message created

		//our names for these fields
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyMsg:   "call-stack-message",
			logrus.FieldKeyLevel: "severity",
		},
	}
}

func (ls LogSet) configureKafka(cfg Config) error {
	if !cfg.useKafka() {
		reportKafkaConfig(nil, cfg, ls)
		return nil
	}

	sink, err := newLiveKafkaSink("kafkahook",
		cfg.getKafkaLevel(),
		logrusFormatter(),
		cfg.getBrokers(),
		cfg.Kafka.Topic,
		false)

	// One cause of errors: can't reach any brokers
	// c.f. https://github.com/Shopify/sarama/blob/master/client.go#L114
	if err != nil {
		return err
	}
	reportKafkaConfig(sink, cfg, ls)

	ls.dumpBundle.replaceKafka(sink)

	return nil
}

func (ls LogSet) configureGraphite(cfg Config) error {
	var gCfg *graphite.Config

	if cfg.useGraphite() {
		addr, err := net.ResolveTCPAddr("tcp", cfg.getGraphiteServer())
		if err != nil {
			return err
		}

		gCfg = &graphite.Config{
			Addr:          addr,
			Registry:      ls.metrics,
			FlushInterval: 30 * time.Second,
			DurationUnit:  time.Nanosecond,
			Prefix:        "sous",
			Percentiles:   []float64{0.5, 0.75, 0.95, 0.99, 0.999},
		}

	}
	reportGraphiteConfig(gCfg, ls)
	ls.dumpBundle.graphiteConfig = gCfg

	gCtx, cancel := context.WithCancel(ls.context)

	if ls.graphiteCancel != nil {
		ls.graphiteCancel()
	}

	ls.graphiteCancel = cancel
	go metricsLoop(gCtx, ls, gCfg)

	return nil
}

func metricsLoop(ctx context.Context, ls LogSet, cfg *graphite.Config) {
	interval := time.Second * 30
	if cfg != nil {
		interval = cfg.FlushInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// TODO: metrics observation goes here
			if cfg != nil {
				if err := graphite.Once(*cfg); err != nil {
					reportGraphiteError(ls, err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// Metrics returns a MetricsSink, which can receive various metrics related method calls. (c.f)
// LogSet.Metrics returns itself -
// xxx quickie for providing metricssink
func (ls LogSet) Metrics() MetricsSink {
	return ls
}

// Done signals that the LogSet (as a MetricsSink) is done being used -
// LogSet's current implementation treats this as a no-op but c.f. MetricsSink.
// xxx noop until extracted a metrics sink
func (ls LogSet) Done() {
}

// Console implements LogSink on LogSet
func (ls LogSet) Console() WriteDoner {
	return nopDoner(ls.err)
}

// ExtraConsole implements LogSink on LogSet
func (ls LogSet) ExtraConsole() WriteDoner {
	return nopDoner(ls.extraConsole)
}

func (ls LogSet) imposeLevel() {
	ls.logrus.SetLevel(logrus.ErrorLevel)

	if ls.level >= WarningLevel {
		ls.logrus.SetLevel(logrus.WarnLevel)
	}

	if ls.level >= DebugLevel {
		ls.logrus.SetLevel(logrus.DebugLevel)
	}

	if ls.level >= ExtraDebug1Level {
		ls.logrus.SetLevel(logrus.DebugLevel)
	}
}

// GetLevel returns the current level of this LogSet
func (ls LogSet) GetLevel() Level {
	return ls.level
}

// BeSilent not only sets the log level to Error,
// it also sets console output to Discard
//.Note that as implemented, console output cannot be recovered -
// it's assumed that BeSilent will be called once per execution.
func (ls *LogSet) BeSilent() {
	ls.level = 0
	ls.imposeLevel()
	ls.dumpBundle.err = ioutil.Discard
}

// BeQuiet gets the LogSet to discard all its output
func (ls *LogSet) BeQuiet() {
	ls.level = CriticalLevel
	ls.imposeLevel()
}

// BeTerse gets the LogSet to print debugging output
func (ls *LogSet) BeTerse() {
	ls.level = WarningLevel
	ls.imposeLevel()
}

// BeHelpful gets the LogSet to print debugging output
func (ls *LogSet) BeHelpful() {
	ls.level = DebugLevel
	ls.imposeLevel()
}

// BeChatty gets the LogSet to print all its output - useful for temporary debugging
func (ls *LogSet) BeChatty() {
	ls.level = ExtraDebug1Level
	ls.imposeLevel()
}
