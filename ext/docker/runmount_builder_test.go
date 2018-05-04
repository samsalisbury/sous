package docker

import (
	"fmt"
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

func TestRunmountBuilder_Run(t *testing.T) {
	sh, _ := shell.Default()
	ctx := sous.BuildContext{
		Sh: sh,
	}
	err := run(ctx, "193fede9eafd")
	if err != nil {
		fmt.Println("err : ", err)
	}
	assert.FailNow(t, "")
}

func TestRunmountBuilder_ExtractRunSpec(t *testing.T) {
	path
}
