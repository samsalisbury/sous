package docker

import (
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

func TestRunnableBuilder_ExtractFiles(t *testing.T) {
	sh, ctl := shell.NewTestShell()

	builder := runnableBuilder{
		RunSpec: SplitImageRunSpec{
			Files: []sbmInstall{{
				Source:      sbmFile{"from"},
				Destination: sbmFile{"to"},
			}},
		},
		splitBuilder: &splitBuilder{
			context: &sous.BuildContext{
				Sh: sh,
			},
		},
	}

	assert.NoError(t, builder.extractFiles())
	assert.Len(t, ctl.CmdsLike("docker", "cp"), 1)
}

func TestRunnableBuilder_Build(t *testing.T) {
	sh, ctl := shell.NewTestShell()

	builder := runnableBuilder{
		RunSpec: SplitImageRunSpec{
			Files: []sbmInstall{{
				Source:      sbmFile{"from"},
				Destination: sbmFile{"to"},
			}},
		},
		splitBuilder: &splitBuilder{
			context: &sous.BuildContext{
				Sh: sh,
			},
		},
	}

	_, cctl := ctl.CmdFor("docker", "build")
	cctl.ResultSuccess("Successfully built cabba9edeadbeef", "")

	assert.NoError(t, builder.build())
	assert.Len(t, ctl.CallsTo("CD"), 1)
	assert.Len(t, ctl.CmdsLike("docker", "build"), 1)
}
