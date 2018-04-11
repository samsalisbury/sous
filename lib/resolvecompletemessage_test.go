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
	startSecs := int64(math.Pow(10, 1000))
	status := &ResolveStatus{
		Started:  time.Unix(startSecs, 0),
		Finished: time.Unix(startSecs+3000, 0),
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
	logCalls := control.CallsTo("Fields")
	require.Len(t, logCalls, 1)
	msgs := logCalls[0].PassedArgs().Get(0).([]logging.EachFielder)

	fixedFields := map[string]interface{}{
		"@loglov3-otl":       logging.SousResolutionResultV1,
		"severity":           logging.WarningLevel,
		"call-stack-message": "Recording stable status",
		"error-count":        1,
	}

	logging.AssertMessageFieldlist(t, msgs, append(logging.StandardVariableFields, logging.IntervalVariableFields...), fixedFields)
}
