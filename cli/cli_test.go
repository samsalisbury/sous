package cli

import (
	"bytes"
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestInvokeRectifyWithDebugFlags(t *testing.T) {
	assert := assert.New(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	c, err := NewSousCLI(semv.MustParse(`1.2.3`), stdout, stderr)
	assert.NoError(err)

	exe, err := c.Prepare([]string{`sous`, `rectify`, `-d`, `-v`, `-all`})
	assert.NoError(err)
	assert.Len(exe.Args, 0)
}
