package messages

import (
	"encoding/json"
	"fmt"
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
		if encountered[elements[v]] == true {
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
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	} else {
		return t.Name()
	}
}

//DefaultStructInfoFunc is the default implementation for structs to use to return names, types, and jsonStruct
//It checks if the interface passed in implements InnerLogger and will use that instead
func DefaultStructInfo(o interface{}) (names []string, types []string, jsonStruct string) {

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
				innerNames, innerTypes, _ := DefaultStructInfo(f.Value())
				names = append(names, innerNames...)
				types = append(types, innerTypes...)
			}
			//fmt.Printf("value   : %+v\n", f.Value())
			//fmt.Printf("is zero : %+v\n", f.IsZero())
			//fmt.Printf("is kind : %s\n", f.Kind().String())
			//fmt.Printf("is type : %s\n", getType(f.Value()))
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

type LogFieldsMessage struct {
	logging.CallerInfo
	logging.Level
	Fields             []string
	Types              []string
	JSONRepresentation string
	jsonObj            *gabs.Container
	msg                string
}

func ReportLogFieldsMessage(msg string, loglvl logging.Level, items ...interface{}) LogFieldsMessage {
	logMessage := LogFieldsMessage{
		CallerInfo:         logging.GetCallerInfo(logging.NotHere()),
		Level:              loglvl,
		Fields:             []string{},
		Types:              []string{},
		JSONRepresentation: "",
		msg:                msg,
	}
	logMessage.jsonObj = gabs.New()

	for _, item := range items {
		fields, types, jsonRep := DefaultStructInfo(item)
		logMessage.addFields(fields...)
		logMessage.addTypes(types...)
		logMessage.addJSON(jsonRep)
	}
	return logMessage
}

func (l *LogFieldsMessage) addJSON(json string) {
	if l.jsonObj == nil {
		l.jsonObj = gabs.New()
	}
	l.jsonObj.Set(json, "message")
}
func (l *LogFieldsMessage) addFields(fields ...string) {
	if l.Fields == nil {
		l.Fields = []string{}
	}
	l.Fields = append(l.Fields, fields...)
}
func (l *LogFieldsMessage) DefaultLevel() logging.Level {
	panic("not implemented")
}
func (l *LogFieldsMessage) addTypes(types ...string) {
	if l.Types == nil {
		l.Types = []string{}
	}
	l.Types = append(l.Types, types...)
}
func (l *LogFieldsMessage) Message() string {
	return l.msg
}

func (l *LogFieldsMessage) EachField(fn logging.FieldReportFn) {

	fn("@loglov3-otl", "sous-generic-v1")
	fn("fields", strings.Join(l.Fields, ","))
	fn("types", strings.Join(l.Types, ","))
	if l.jsonObj != nil {
		fn("jsonStruct", l.jsonObj.String())

	}
	l.CallerInfo.EachField(fn)

}
