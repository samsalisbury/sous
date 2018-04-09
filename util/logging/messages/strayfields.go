package messages

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Jeffail/gabs"
	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/structs"
	"github.com/opentable/sous/util/logging"
)

type strayFields struct {
	fields  []string
	types   []string
	withIDs bool
	ids     []string
	values  []string
	jsonObj *gabs.Container
}

func assembleStrayFields(withIDs bool, items ...interface{}) strayFields {
	sf := &strayFields{withIDs: withIDs}
	sf.jsonObj = gabs.New()
	if _, err := sf.jsonObj.Array("message", "array"); err != nil {
		fmt.Println("Failed to add object array: ", err.Error())
	}
	for _, item := range items {
		sf.addItem(item)
	}
	return *sf
}

// EachField defines EachFielder on strayFields
func (sf strayFields) EachField(fn logging.FieldReportFn) {
	fn(logging.SousFields, strings.Join(removeDuplicates(sf.fields), ","))
	fn(logging.SousTypes, strings.Join(removeDuplicates(sf.types), ","))

	if sf.withIDs {
		fn(logging.SousIds, strings.Join(removeDuplicates(sf.ids), ","))
		fn(logging.SousIdValues, strings.Join(removeDuplicates(sf.values), ","))
	}

	if sf.jsonObj != nil {
		if n, err := sf.jsonObj.ArrayCount("message", "array"); n > 0 && err == nil {
			fn(logging.JsonValue, sf.jsonObj.String())
		}
	}
}

func (sf *strayFields) addItem(item interface{}) {
	fs, ts, jsonRep := defaultStructInfo(item)
	sf.fields = append(sf.fields, fs...)
	sf.types = append(sf.types, ts...)
	sf.addJSON(jsonRep)

	if sf.withIDs {
		sf.extractID(item)
	}
}

func (sf *strayFields) addJSON(json string) {
	if sf.jsonObj == nil {
		sf.jsonObj = gabs.New()
		if _, err := sf.jsonObj.Array("message", "array"); err != nil {
			fmt.Println("error:", err)
		}
	}
	if err := sf.jsonObj.ArrayAppend(json, "message", "array"); err != nil {
		fmt.Println("error: ", err)
	}
}

func (sf *strayFields) extractID(o interface{}) {
	if structs.IsStruct(o) {
		s := structs.New(o)
		sf.insertID(s.Name(), o)
		for _, f := range s.Fields() {
			if f.IsExported() {
				sf.insertID(f.Name(), f.Value())
			}
		}
	} else {
		if t := reflect.TypeOf(o); t != nil {
			sf.insertID(t.Name(), o)
		}
	}
}

func (sf *strayFields) insertID(idName string, idValue interface{}) {
	if !strings.Contains(strings.ToLower(idName), "id") {
		return
	}
	strIDValue := ""
	if val, ok := idValue.(string); ok {
		strIDValue = val
	} else {
		strIDValue = spew.Sdump(idValue)
	}
	sf.ids = append(sf.ids, idName)
	sf.values = append(sf.values, strIDValue)
}

func defaultStructInfo(o interface{}, depth ...int) (fields []string, types []string, jsonStruct string) {

	//stop cyclical logging
	currentDepth := 0
	if len(depth) > 0 {
		if depth[0] > 10 {
			return
		}
		currentDepth = depth[0] + 1
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

func failedToParseJSON(name string) string {
	jsonStruct := fmt.Sprintf("{\"%s\": \"Fail to create json\"}", name)
	return jsonStruct

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

func deserialSpew(o interface{}) (spewString string) {
	spewString = spew.Sdump(o)
	return spewString
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
