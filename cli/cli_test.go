package cli

import (
	"bytes"
	"log"
	"testing"

	"github.com/opentable/sous/util/cmdr"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvokeBareSous(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	c, err := NewSousCLI(semv.MustParse(`1.2.3`), stdout, stderr)
	assert.NoError(err)

	exe, err := c.Prepare([]string{`sous`})
	assert.NoError(err)
	assert.Len(exe.Args, 0)

	var r cmdr.Result
	require.NotPanics(func() {
		r = c.InvokeWithoutPrinting([]string{"sous", "help"})
	})
	log.Printf("%T %v", r, r)
	assert.IsType(cmdr.SuccessResult{}, r)

}

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
	require.NotNil(rect.SourceFlags)
	assert.Equal(rect.SourceFlags.All, true)
	assert.Regexp(`Verbose debugging`, stderr.String())
	assert.Regexp(`Regular debugging`, stderr.String())
}

func TestInvokeBuildWithRepoSelector(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	c, err := NewSousCLI(semv.MustParse(`1.2.3`), stdout, stderr)
	assert.NoError(err)

	exe, err := c.Prepare([]string{`sous`, `build`, `-repo`, `github.com/opentable/sous`})
	require.NoError(err)
	assert.Len(exe.Args, 0)

	build := exe.Cmd.(*SousBuild)

	assert.NotNil(build.Labeller)
	assert.NotNil(build.Registrar)
	assert.Equal(build.DeployFilterFlags.Repo, `github.com/opentable/sous`)

}
