package cli

import (
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
)

func TestInvocationMessage(t *testing.T) {
	msg := newInvocationMessage([]string{"testing", "test"})

	fixedFields := []string{
		"@timestamp",
		"call-stack-file",
		"call-stack-line-number",
		"call-stack-function",
		"thread-name",
	}

	variableFields := map[string]interface{}{
		"@loglov3-otl": "sous-cli-v1",
		"arguments":    `["testing" "test"]`,
	}

	logging.AssertMessageFields(t, msg, fixedFields, variableFields)
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

	variableFields := map[string]interface{}{
		"@loglov3-otl": "sous-cli-v1",
		"arguments":    `["testing" "test"]`,
		"exit-code":    1,
	}

	logging.AssertMessageFields(t, msg, fixedFields, variableFields)
}
