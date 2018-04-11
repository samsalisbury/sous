package logging

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

type entryID struct {
	id, name, uuid string
}

// Fields implements LogSink on LogSet.
func (ls LogSet) Fields(items []EachFielder) {
	logto := logrus.NewEntry(ls.logrus)
	redundantFields := map[FieldName][]interface{}{}
	haveRedundant := false

	var hasOTL, hasLevel, hasMessage bool

	items = append(items, ls.appIdent, ls.entryID())
	level := WarningLevel

	messages := []string{}

	var strays *strayFields

	for _, item := range items {
		if s, is := item.(strayFields); is {
			strays = &s
			continue
		}
		item.EachField(func(name FieldName, value interface{}) {
			switch name {
			default:
				if list, yes := redundantFields[name]; yes {
					redundantFields[name] = append(list, value)
					haveRedundant = true
					return
				}
				redundantFields[name] = []interface{}{}
				if name == Loglov3Otl {
					hasOTL = true
				}
				logto = logto.WithField(string(name), value)
			case Severity:
				newLevel, is := value.(Level)
				if !is {
					return
				}
				hasLevel = true
				if !hasLevel {
					level = newLevel
					return
				}
				if level < newLevel {
					level = newLevel
				}
				messages = append(messages, "Redundant serverity")
			case CallStackMessage:
				hasMessage = true
				messages = append(messages, fmt.Sprintf("%s", value))
			}
		})
	}

	if !hasOTL {
		messages = append(messages, "No OTL provided")
		logto = logto.WithField(string(Loglov3Otl), SousGenericV1)
	}

	if !hasMessage {
		messages = append(messages, "No message provided")
	}

	if !hasLevel {
		messages = append(messages, "No level provided")
	}

	if haveRedundant {
		if strays == nil {
			s := assembleStrayFields()
			strays = &s
		}
		strays.addRedundants(redundantFields)
	}

	if strays != nil {
		strays.EachField(func(name FieldName, value interface{}) {
			logto.WithField(string(name), value)
		})
	}

	switch level {
	default:
		logto.Printf("unknown Level: %d - %q", level, strings.Join(messages, "\n"))
	case CriticalLevel:
		logto.Error(strings.Join(messages, "\n"))
	case WarningLevel:
		logto.Warn(strings.Join(messages, "\n"))
	case InformationLevel:
		logto.Info(strings.Join(messages, "\n"))
	case DebugLevel:
		logto.Debug(strings.Join(messages, "\n"))
	case ExtraDebug1Level:
		logto.Debug(strings.Join(messages, "\n"))
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
