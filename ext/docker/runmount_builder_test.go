package docker

import (
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

func TestRunmountBuilder_Build(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "build")
	cctl.ResultSuccess("Successfully built cabba9edeadbeef", "")
	ctx := sous.BuildContext{
		Sh: sh,
	}
	buildID, _ := build(ctx)
	assert.Equal(t, "cabba9edeadbeef", buildID)
}
