package messages

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/Jeffail/gabs"
	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/structs"
	"github.com/opentable/sous/util/logging"
)

//InnerLogger interface is used if struct wants to provide it's own way of returns fields, types, and json string
type InnerLogger interface {
	InnerLogInfo() (fields []string, types []string, jsonStruct string)
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
	if myvar != nil {
		var t reflect.Type
		if t = reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
			return "*" + t.Elem().Name()
		}
		return t.Name()
	}
	return ""
}

func failedToParseJSON(name string) string {
	jsonStruct := fmt.Sprintf("{\"%s\": \"Fail to create json\"}", name)
	return jsonStruct

}

func (l *logFieldsMessage) insertID(idName string, idValue interface{}) {
	if strings.Contains(strings.ToLower(idName), "id") {
		strIDValue := ""
		if val, ok := idValue.(string); ok {
			strIDValue = val
		} else {
			strIDValue = spew.Sdump(idValue)
		}
		if val, ok := l.idsMap[idName]; !ok {
			l.idsMap[idName] = strIDValue
		} else {
			if !strings.Contains(val, strIDValue) {
				l.idsMap[idName] = val + ", " + strIDValue
			}
		}
	}
}

func (l *logFieldsMessage) extractID(o interface{}) {
	if l.withIDs {
		if structs.IsStruct(o) {
			s := structs.New(o)
			l.insertID(s.Name(), o)
			for _, f := range s.Fields() {
				if f.IsExported() {
					l.insertID(f.Name(), f.Value())
				}
			}
		} else {
			if t := reflect.TypeOf(o); t != nil {
				l.insertID(t.Name(), o)
			}
		}
	}
}

//DefaultStructInfo is the default implementation for structs to use to return fields, types, and jsonStruct
//It checks if the interface passed in implements InnerLogger and will use that instead
func defaultStructInfo(o interface{}, depth ...int) (fields []string, types []string, jsonStruct string) {

	//stop cyclical logging
	currentDepth := 0
	if len(depth) > 0 {
		if depth[0] > 10 {
			return
		}
		currentDepth = depth[0] + 1
	}

	if innerLog, ok := o.(InnerLogger); ok {
		fields, types, jsonStruct = innerLog.InnerLogInfo()
		return
	}
	v := reflect.ValueOf(o)

	// if pointer get the underlying element≤
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	//handle when it's not a struct
	if v.Kind() != reflect.Struct {
		fields = []string{}
		types = []string{}
		oType := getType(o)
		types = append(types, oType)
		jsonObj := gabs.New()
		if _, err := jsonObj.Set(o, oType, oType); err != nil {
			jsonStruct = failedToParseJSON(oType)
		} else {
			jsonStruct = jsonObj.String()
		}
		return
	}

	//handle error interface explicitly to extract error msg
	if anErr, ok := o.(error); ok {
		fields = []string{}
		types = []string{"error"}

		jsonObj := gabs.New()
		if _, err := jsonObj.Set(anErr.Error(), "error", "error"); err != nil {
			jsonStruct = failedToParseJSON("error")
			return
		}
		jsonStruct = jsonObj.String()
		return
	}

	s := structs.New(o)

	fields = s.Names()
	fields = append(fields, s.Name())
	types = []string{}

	types = append(types, getType(o))

	for _, f := range s.Fields() {
		if f.IsExported() {
			types = append(types, getType(f.Value()))
			if f.Kind() == reflect.Struct {
				innerNames, innerTypes, _ := defaultStructInfo(f.Value(), currentDepth)
				fields = append(fields, innerNames...)
				types = append(types, innerTypes...)
			}
		}
	}

	jsonStruct = deserialSpew(o)

	return fields, types, jsonStruct
}

func deserialSpew(o interface{}) (spewString string) {
	spewString = spew.Sdump(o)
	return spewString
}

type logFieldsMessage struct {
	logging.CallerInfo
	logging.Level
	submessages        []logging.EachFielder
	Fields             []string
	Types              []string
	JSONRepresentation string
	jsonObj            *gabs.Container
	msg                string
	console            bool
	serverConsole      bool
	withIDs            bool
	idsMap             map[string]string
}

func (l logFieldsMessage) WriteToConsole(console io.Writer) {
	if l.console {
		fmt.Fprintf(console, "%s\n", l.composeMsg())
	}
}

func (l logFieldsMessage) WriteExtraToConsole(console io.Writer) {
	if l.console {
		if l.jsonObj != nil {
			fmt.Fprintf(console, "%s\n", l.jsonObj.StringIndent("", " "))
		}

		fmt.Fprintf(console, "Fields: %s\n", strings.Join(l.Fields, ","))
		fmt.Fprintf(console, "Types: %s\n", strings.Join(l.Types, ","))
	}
}

func (l logFieldsMessage) returnIDs() (ids string, values string) {

	idsSlice := []string{}
	valuesSlice := []string{}

	if l.withIDs {

		for k := range l.idsMap {
			idsSlice = append(idsSlice, k)
		}
		sort.Strings(idsSlice)

		for _, k := range idsSlice {
			valuesSlice = append(valuesSlice, l.idsMap[k])
		}
	}

	ids = strings.Join(idsSlice, ",")
	values = strings.Join(valuesSlice, ",")

	return ids, values
}

func (l logFieldsMessage) composeMsg() string {
	return l.msg
}
func buildLogFieldsMessage(msg string, console bool, withIDs bool, loglvl logging.Level) logFieldsMessage {
	logMessage := logFieldsMessage{
		CallerInfo:         logging.GetCallerInfo(logging.NotHere()),
		Level:              loglvl,
		Fields:             []string{},
		Types:              []string{},
		JSONRepresentation: "",
		msg:                msg,
		console:            console,
		withIDs:            withIDs,
	}

	logMessage.idsMap = make(map[string]string)
	logMessage.jsonObj = gabs.New()
	if _, err := logMessage.jsonObj.Array("message", "array"); err != nil {
		fmt.Println("Failed to add object array: ", err.Error())
	}
	return logMessage

}

//ReportLogFieldsMessageWithIDs report message with Ids
func ReportLogFieldsMessageWithIDs(msg string, loglvl logging.Level, logSink logging.LogSink, items ...interface{}) {
	logMessage := buildLogFieldsMessage(msg, false, true, loglvl)
	logMessage.CallerInfo.ExcludeMe()
	logMessage.reportLogFieldsMessage(logSink, items...)

}

//ReportLogFieldsMessageToConsole report message to console
func ReportLogFieldsMessageToConsole(msg string, loglvl logging.Level, logSink logging.LogSink, items ...interface{}) {
	logMessage := buildLogFieldsMessage(msg, true, false, loglvl)
	logMessage.CallerInfo.ExcludeMe()
	logMessage.reportLogFieldsMessage(logSink, items...)
}

//ReportLogFieldsMessage generate a logFieldsMessage log entry
func ReportLogFieldsMessage(msg string, loglvl logging.Level, logSink logging.LogSink, items ...interface{}) {
	logMessage := buildLogFieldsMessage(msg, false, true, loglvl)
	logMessage.CallerInfo.ExcludeMe()

	logMessage.reportLogFieldsMessage(logSink, items...)
}

func (l logFieldsMessage) reportLogFieldsMessage(logSink logging.LogSink, items ...interface{}) {
	l.CallerInfo.ExcludeMe()

	for _, item := range items {
		if sm, is := item.(logging.EachFielder); is {
			l.submessages = append(l.submessages, sm)
		}
		l.extractID(item)
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
		if _, err := l.jsonObj.Array("message", "array"); err != nil {
			fmt.Println("error:", err)
		}
	}
	if err := l.jsonObj.ArrayAppend(json, "message", "array"); err != nil {
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

//EachField will make sure individual fields are added for OTL
func (l logFieldsMessage) EachField(fn logging.FieldReportFn) {

	fn("sous-fields", strings.Join(removeDuplicates(l.Fields), ","))
	fn("sous-types", strings.Join(removeDuplicates(l.Types), ","))

	if l.withIDs {
		ids, values := l.returnIDs()
		fn("sous-ids", ids)
		fn("sous-id-values", values)
	}

	if l.jsonObj != nil {
		if n, err := l.jsonObj.ArrayCount("message", "array"); n > 0 && err == nil {
			fn("json-value", l.jsonObj.String())
		}

	}
	l.CallerInfo.EachField(fn)

	for _, sm := range l.submessages {
		sm.EachField(fn)
	}

	//In case anyone override the otl field with submessages.  Adding it at the end
	fn("@loglov3-otl", "sous-generic-v1")
}
