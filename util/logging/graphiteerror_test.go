package logging

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphiteError(t *testing.T) {
	spy, ctrl := NewLogSinkSpy()

	reportGraphiteError(spy, fmt.Errorf("something bad"))

	calls := ctrl.CallsTo("Fields")
	if assert.Len(t, calls, 1) {
		call := calls[0]
		items := call.PassedArgs()[0].([]EachFielder)
		if assert.Len(t, items, 3) {
			assert.IsType(t, graphiteError{}, items[2])
		}
	}

	// actually testing that graphiteError fulfills LogMessage
	// if it doesn't, this won't compile
	lm := LogMessage(graphiteError{})
	assert.NotNil(t, lm)
}
