package logging

import (
	"testing"

	"github.com/samsalisbury/semv"
)

func TestApplicationId(t *testing.T) {
	apid := collectAppID(semv.MustParse("2.3.9"), map[string]string{
		"OT_ENV":          "staging-luna",
		"OT_ENV_TYPE":     "staging",
		"OT_ENV_LOCATION": "luna",
		"TASK_ID":         "cabbage",
		"INSTANCE_NO":     "17",
	})

	variableFields := []string{"host"}

	fixedFields := map[string]interface{}{
		"ot-env":              "staging-luna",
		"application-version": "2.3.9",
		"ot-env-type":         "staging",
		"ot-env-location":     "luna",
		"singularity-task-id": "cabbage",
		"service-type":        "sous",
		"instance-no":         uint(17),
		"sequence-number":     uint(1), // a little fragile
	}

	AssertMessageFields(t, apid, variableFields, fixedFields)
}
