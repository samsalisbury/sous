package logging

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphiteError(t *testing.T) {
	spy, ctrl := NewLogSinkSpy()

	reportGraphiteError(spy, fmt.Errorf("something bad"))

	calls := ctrl.CallsTo("LogMessage")
	assert.Len(t, calls, 1)
	call := calls[0]
	assert.IsType(t, graphiteError{}, call.PassedArgs()[1])

	// actually testing that graphiteError fulfills LogMessage
	// if it doesn't, this won't compile
	lm := LogMessage(graphiteError{})
	assert.NotNil(t, lm)
}
