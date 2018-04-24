package logging

import "testing"

func TestStrayFields(t *testing.T) {
	sf := assembleStrayFields(42)
	sf.addRedundants(map[FieldName][]interface{}{
		Severity: []interface{}{WarningLevel, DebugLevel},
	})
	rawAssertMessageFields(t, []EachFielder{sf}, []string{}, map[string]interface{}{
		"sous-types": "int",
		"json-value": "{\"message\":{\"array\":[\"{\\\"int\\\":{\\\"int\\\":42}}\"],\"redundant\":{\"severity\":[1,3]}}}",
	})

}
