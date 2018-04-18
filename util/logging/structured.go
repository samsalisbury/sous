package logging

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

type (
	redundantFields struct {
		fs   map[FieldName][]interface{}
		have bool
	}

	entryID struct {
		id, name, uuid string
	}
)

func (r *redundantFields) check(n FieldName, v interface{}) bool {
	if list, yes := r.fs[n]; yes {
		r.fs[n] = append(list, v)
		r.have = true
		return true
	}
	r.fs[n] = []interface{}{}
	return false
}

func (r *redundantFields) softCheck(n FieldName, v interface{}) bool {
	if _, yes := r.fs[n]; yes {
		return true
	}
	r.fs[n] = []interface{}{}
	return false
}

func (r redundantFields) extra(n FieldName) bool {
	vs, have := r.fs[n]
	if !have {
		return false
	}

	return len(vs) > 0
}

func (r redundantFields) any(n FieldName) bool {
	_, have := r.fs[n]
	return have
}

// Fields implements LogSink on LogSet.
func (ls LogSet) Fields(items []EachFielder) {
	logto := logrus.NewEntry(ls.logrus)
	redundants := redundantFields{fs: map[FieldName][]interface{}{}}

	items = append(items, ls.appIdent, ls.entryID())
	level := ExtraDebug1Level

	messages := []string{}

	var strays *strayFields

	for _, item := range items {
		if s, is := item.(strayFields); is {
			strays = &s
			continue
		}
		item.EachField(func(name FieldName, value interface{}) {
			isRedundant := redundants.check(name, value)

			switch name {
			default:
				if isRedundant {
					return
				}

				logto = logto.WithField(string(name), value)
			case Severity:
				newLevel, is := value.(Level)
				if !is {
					return
				}
				if newLevel < level {
					level = newLevel
				}
			case CallStackMessage:
				messages = append(messages, fmt.Sprintf("%s", value))
			}
		})
	}

	for _, item := range ls.ctxFields {
		item.EachField(func(name FieldName, value interface{}) {
			// context fields get overridden
			if redundants.softCheck(name, value) {
				return
			}
			logto = logto.WithField(string(name), value)
		})
	}

	if !redundants.any(Loglov3Otl) {
		messages = append(messages, "No OTL provided")
		logto = logto.WithField(string(Loglov3Otl), SousGenericV1)
	}

	if !redundants.any(CallStackMessage) {
		messages = append(messages, "No message provided")
	}

	if !redundants.any(Severity) {
		messages = append(messages, "No level provided")
		level = WarningLevel
	}

	if redundants.have {
		if redundants.extra(Severity) {
			messages = append(messages, "Redundant severities")
		}
		if strays == nil {
			s := assembleStrayFields()
			strays = &s
		}
		strays.addRedundants(redundants.fs)
	}

	if strays != nil {
		strays.EachField(func(name FieldName, value interface{}) {
			logto = logto.WithField(string(name), value)
		})
	}

	message := strings.Join(messages, "\n")

	logto.Message = message
	err := ls.dumpBundle.sendToKafka(level, logto)
	if err != nil {
		Deliver(ls, Console(err)) //won't re-enter Fields
	}

	switch level {
	default:
		logto.Printf("unknown Level: %d - %q", level, message)
	case CriticalLevel:
		logto.Error(message)
	case WarningLevel:
		logto.Warn(message)
	case InformationLevel:
		logto.Info(message)
	case DebugLevel:
		logto.Debug(message)
	case ExtraDebug1Level:
		logto.Debug(message)
	}

}

func (ls LogSet) entryID() entryID {
	id := entryID{
		id:   "sous",
		name: ls.name,
		uuid: uuid.New(),
	}

	if ls.appRole != "" {
		id.id = "sous-" + ls.appRole
	}

	return id
}

func (id entryID) EachField(f FieldReportFn) {
	f(ComponentId, id.id)
	f(LoggerName, id.name)
	f(Uuid, id.uuid)
}

func enforceSchema(name FieldName, val interface{}) {
	if false {
		panic("bad logging")
	}
}
