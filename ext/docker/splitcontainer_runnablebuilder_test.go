package docker

import (
	"bytes"
	"strings"
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

func TestRunnableBuildpackBuildTemplating(t *testing.T) {
	sb := &runnableBuilder{
		RunSpec: SplitImageRunSpec{
			Image: sbmImage{From: "scratch"},
			Files: []sbmInstall{
				{Source: sbmFile{Dir: "src"}, Destination: sbmFile{Dir: "dest"}},
			},
			Exec: []string{"cat", "/etc/shadow"},
		},
		splitBuilder: &splitBuilder{
			context: &sous.BuildContext{
				Source: sous.SourceContext{
					RemoteURL:  "github.com/example/project",
					OffsetDir:  "",
					NearestTag: sous.Tag{Name: "1.2.3"},
					Revision:   "cabba9edeadbeef",
				},
			},
			//VersionConfig:  "APP_VERSION=1.2.3",
			//RevisionConfig: "APP_REVISION=cabba9edeadbeef",
		},
	}
	buf := &bytes.Buffer{}

	err := sb.templateDockerfileBytes(buf)
	if err != nil {
		t.Error(err)
	}
	dockerfile := buf.String()
	hasString := func(needle string) {
		if strings.Index(dockerfile, needle) == -1 {
			t.Errorf("No %q in dockerfile.", needle)
		}
	}
	hasString("FROM scratch")
	hasString("ENV APP_VERSION=1.2.3 APP_REVISION=cabba9edeadbeef")
	hasString("COPY dest dest")
	hasString(`CMD ["cat","/etc/shadow"]`)
	//hasString("LABEL com.opentable.sous.build-image=") //once we push the build image...
}
