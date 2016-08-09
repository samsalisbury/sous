package cli

import (
	"bytes"
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeRectifyWithDebugFlags(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	c, err := NewSousCLI(semv.MustParse(`1.2.3`), stdout, stderr)
	assert.NoError(err)

	exe, err := c.Prepare([]string{`sous`, `rectify`, `-d`, `-v`, `-all`})
	assert.NoError(err)
	assert.Len(exe.Args, 0)
	require.IsType(&SousRectify{}, exe.Cmd)

	rect := exe.Cmd.(*SousRectify)

	assert.NotNil(rect.Config)
	assert.NotNil(rect.DockerClient)
	assert.NotNil(rect.Deployer)
	assert.NotNil(rect.Registry)
	assert.NotNil(rect.GDM)
	require.NotNil(rect.flags)
	assert.Equal(rect.flags.all, true)
}
