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

func (l *fallbackLogger) Child(name string, ctx ...logging.EachFielder) logging.LogSink {
	return l
}

func (l *fallbackLogger) Fields(items []logging.EachFielder) {
	fmt.Printf("Log entry:\n")
	for _, i := range items {
		i.EachField(func(n logging.FieldName, v interface{}) {
			fmt.Printf("%s %#v\n", n, v)
		})
	}
}

func (l *fallbackLogger) Metrics() logging.MetricsSink {
	panic("not implemented")
}

func (l *fallbackLogger) Console() logging.WriteDoner {
	panic("not implemented")
}

func (l *fallbackLogger) ExtraConsole() logging.WriteDoner {
	panic("not implemented")
}

func (l *fallbackLogger) AtExit() {}

func (l fallbackLogger) ForceDefer() bool {
	return false
}
