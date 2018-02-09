package messages

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/Jeffail/gabs"
	"github.com/fatih/structs"
	"github.com/opentable/sous/util/logging"
)

//InnerLogger interface is used if struct wants to provide it's own way of returns names, types, and json string
type InnerLogger interface {
	InnerLogInfo() (names []string, types []string, jsonStruct string)
}

func removeDuplicates(elements []string) []string {
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		//don't include empty
		if elements[v] == "" {
			continue
		}
		if encountered[elements[v]] {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}

func getType(myvar interface{}) string {
	var t reflect.Type
	if t = reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	}
	return t.Name()
}

//DefaultStructInfo is the default implementation for structs to use to return names, types, and jsonStruct
//It checks if the interface passed in implements InnerLogger and will use that instead
func defaultStructInfo(o interface{}) (names []string, types []string, jsonStruct string) {

	if innerLog, ok := o.(InnerLogger); ok {
		names, types, jsonStruct = innerLog.InnerLogInfo()
		return
	}

	s := structs.New(o)

	names = s.Names()
	names = append(names, s.Name())
	types = []string{}

	types = append(types, getType(o))

	for _, f := range s.Fields() {
		if f.IsExported() {
			types = append(types, getType(f.Value()))
			if f.Kind() == reflect.Struct {
				innerNames, innerTypes, _ := defaultStructInfo(f.Value())
				names = append(names, innerNames...)
				types = append(types, innerTypes...)
			}
		}
	}

	jsonStruct = ""
	mapParent := structs.Map(o)
	if mapB, err := json.Marshal(mapParent); err == nil {
		jsonStruct = string(mapB)
	} else {
		fmt.Println("Failure to marshal map", err.Error())
	}
	names = removeDuplicates(names)
	types = removeDuplicates(types)
	return
}

type logFieldsMessage struct {
	logging.CallerInfo
	logging.Level
	Fields             []string
	Types              []string
	JSONRepresentation string
	jsonObj            *gabs.Container
	msg                string
	console            bool
}

func (l logFieldsMessage) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "%s\n", l.composeMsg())
	if l.jsonObj != nil {
		fmt.Fprintf(console, "%s\n", l.jsonObj.StringIndent("", " "))
	}
	fmt.Fprintf(console, "Fields: %s\n", strings.Join(l.Fields, ","))
	fmt.Fprintf(console, "Types: %s\n", strings.Join(l.Types, ","))
}

func (l logFieldsMessage) composeMsg() string {
	return l.msg
}
func buildLogFieldsMessage(msg string, console bool, loglvl logging.Level) logFieldsMessage {
	logMessage := logFieldsMessage{
		CallerInfo:         logging.GetCallerInfo(logging.NotHere()),
		Level:              loglvl,
		Fields:             []string{},
		Types:              []string{},
		JSONRepresentation: "",
		msg:                msg,
		console:            console,
	}

	return logMessage

}
func ReportLogFieldsMessageToConsole(msg string, loglvl logging.Level, logSink logging.LogSink, items ...interface{}) {
	logMessage := buildLogFieldsMessage(msg, true, loglvl)
	logMessage.CallerInfo.ExcludeMe()
	logMessage.reportLogFieldsMessage(logSink, items...)

}

//ReportLogFieldsMessage generate a logFieldsMessage log entry
func ReportLogFieldsMessage(msg string, loglvl logging.Level, logSink logging.LogSink, items ...interface{}) {
	logMessage := buildLogFieldsMessage(msg, false, loglvl)
	logMessage.CallerInfo.ExcludeMe()

	logMessage.reportLogFieldsMessage(logSink, items...)
}

func (l logFieldsMessage) reportLogFieldsMessage(logSink logging.LogSink, items ...interface{}) {
	l.CallerInfo.ExcludeMe()
	l.jsonObj = gabs.New()

	for _, item := range items {
		fields, types, jsonRep := defaultStructInfo(item)
		l.addFields(fields...)
		l.addTypes(types...)
		l.addJSON(jsonRep)
	}
	logging.Deliver(l, logSink)
}

func (l *logFieldsMessage) addJSON(json string) {
	if l.jsonObj == nil {
		l.jsonObj = gabs.New()
	}
	if _, err := l.jsonObj.Set(json, "message"); err != nil {
		fmt.Println("error: ", err)
	}
}
func (l *logFieldsMessage) addFields(fields ...string) {
	if l.Fields == nil {
		l.Fields = []string{}
	}
	l.Fields = append(l.Fields, fields...)
}

//DefaultLevel return the default log level for this message
func (l logFieldsMessage) DefaultLevel() logging.Level {
	return l.Level
}

func (l *logFieldsMessage) addTypes(types ...string) {
	if l.Types == nil {
		l.Types = []string{}
	}
	l.Types = append(l.Types, types...)
}

//Message return the message string associate with message
func (l logFieldsMessage) Message() string {
	return l.composeMsg()
}

func (l logFieldsMessage) EachField(fn logging.FieldReportFn) {

	fn("@loglov3-otl", "sous-generic-v1")
	fn("fields", strings.Join(l.Fields, ","))
	fn("types", strings.Join(l.Types, ","))
	if l.jsonObj != nil {
		fn("jsonStruct", l.jsonObj.String())

	}
	l.CallerInfo.EachField(fn)

}
