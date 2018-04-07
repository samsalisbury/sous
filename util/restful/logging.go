package restful

import (
	"fmt"

	"github.com/opentable/sous/util/logging"
)

type (
	logSet interface {
	}

	silentLogSet   struct{}
	fallbackLogger struct{}
)

// PlaceholderLogger returns a log set that fulfills the restful logging
// interface - a convenience for when you don't want or need to wrap a logger
// in appropriate interface fulfillment.
func PlaceholderLogger() logSet {
	return &silentLogSet{}
}

func (l *fallbackLogger) Child(name string) LogSink {
	return l
}

func (l *fallbackLogger) LogMessage(lvl Level, msg LogMessage) {
	fmt.Printf("%s %#v\n", lvl, msg)
}

func (l *fallbackLogger) Metrics() MetricsSink {
	panic("not implemented")
}

func (l *fallbackLogger) Console() WriteDoner {
	panic("not implemented")
}

func (l *fallbackLogger) ExtraConsole() WriteDoner {
	panic("not implemented")
}

func (l *fallbackLogger) AtExit() {}
