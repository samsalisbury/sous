package logging

import (
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestApplicationIdMessageFields(t *testing.T) {
	apid := collectAppID(semv.MustParse("2.3.9"), map[string]string{
		"OT_ENV":          "staging-luna",
		"OT_ENV_TYPE":     "staging",
		"OT_ENV_LOCATION": "luna",
		"TASK_ID":         "cabbage",
		"INSTANCE_NO":     "17",
	})

	t.Run("MessageFields", func(t *testing.T) {
		variableFields := []string{"host"}

		fixedFields := map[string]interface{}{
			"ot-env":              "staging-luna",
			"application-version": "2.3.9",
			"ot-env-type":         "staging",
			"ot-env-location":     "luna",
			"singularity-task-id": "cabbage",
			"service-type":        "sous",
			"instance-no":         uint(17),
			"sequence-number":     uint64(1), // a little fragile
		}

		// rawAssertMessageFields because ApplicationID is used to support other
		// messages - it isn't a fully-fledged structured log entry itself
		rawAssertMessageFields(t, []EachFielder{apid}, variableFields, fixedFields)
	})

	t.Run("MetricsScope", func(t *testing.T) {
		assert.Equal(t, "staging.luna", apid.metricsScope())
	})
}
