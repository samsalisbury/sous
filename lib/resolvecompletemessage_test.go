package sous

import (
	"math"
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCompleteMessage(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()
	start_secs := int64(math.Pow(10, 1000))
	status := &ResolveStatus{
		Started:  time.Unix(start_secs, 0),
		Finished: time.Unix(start_secs+3000, 0),
		Phase:    "finished",
		Intended: nil,
		Log:      nil,
		Errs: ResolveErrors{
			Causes: []ErrorWrapper{{
				MarshallableError: MarshallableError{
					Type:   "SomeKindOfError",
					String: "it just all went wrong, okay?",
				},
			}},
		},
	}
	reportResolverStatus(logger, status)

	assert.Len(t, control.Metrics.CallsTo("UpdateTimer"), 1)
	assert.Len(t, control.Metrics.CallsTo("UpdateSample"), 1)
	logCalls := control.CallsTo("LogMessage")
	require.Len(t, logCalls, 1)
	assert.Equal(t, logCalls[0].PassedArgs().Get(0), logging.WarningLevel)
	msg := logCalls[0].PassedArgs().Get(1).(logging.LogMessage)

	fixedFields := map[string]interface{}{
		"error-count":  1,
		"@loglov3-otl": logging.SousResolutionResultV1,
	}

	logging.AssertMessageFields(t, msg, append(logging.StandardVariableFields, logging.IntervalVariableFields...), fixedFields)
}
