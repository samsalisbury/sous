package cli

import (
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
)

func TestInvocationMessage(t *testing.T) {
	msg := newInvocationMessage([]string{"testing", "test"})

	fixedFields := map[string]interface{}{
		"@loglov3-otl": "sous-cli-v1",
		"arguments":    `["testing" "test"]`,
	}

	logging.AssertMessageFields(t, msg, logging.StandardVariableFields, fixedFields)
}

type testResult struct {
	exit int
}

func (t testResult) ExitCode() int {
	return t.exit
}

func TestResultMessage(t *testing.T) {
	msg := newCLIResult(
		[]string{"testing", "test"},
		time.Now(),
		testResult{1},
	)

	/*j
	fixedFields := []string{
		"@timestamp",
		"call-stack-file",
		"call-stack-line-number",
		"call-stack-function",
		"thread-name",
		"duration",
		"started-at",
		"finished-at",
	}
	*/

	fixedFields := map[string]interface{}{
		"@loglov3-otl": "sous-cli-v1",
		"arguments":    `["testing" "test"]`,
		"exit-code":    1,
	}

	logging.AssertMessageFields(t, msg, append(logging.StandardVariableFields, logging.IntervalVariableFields...), fixedFields)
}
