package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogSinkSpy(t *testing.T) {
	lss, _ := newLogSinkSpy()
	assert.Implements(t, (*logSink)(nil), lss)

}
