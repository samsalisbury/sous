package docker

import (
	"fmt"
	"os"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

func createDockerfile(content string) {
	f, err := os.Create("Dockerfile")
	if err != nil {
		fmt.Println("error creating file")
	}
	defer f.Close()
	f.WriteString(content)
}

func removeDockerfile() {
	os.Remove("Dockerfile")
}

func getContext() *sous.BuildContext {
	sh, _ := shell.Default()
	ctx := &sous.BuildContext{
		Sh: sh,
	}
	return ctx
}

func TestRunmountBuildpack_DetectRunmount(t *testing.T) {
	createDockerfile("FROM docker.otenv.com/sous-otj-autobuild-runmount:local")
	defer removeDockerfile()

	rmbp := NewRunmountBuildpack()

	result, _ := rmbp.Detect(getContext())

	assert.True(t, result.Compatible)
}

func TestRunmountBuildpack_DetectNotRunmount(t *testing.T) {
	createDockerfile("FROM docker.otenv.com/sous-otj-autobuild:local")
	defer removeDockerfile()

	rmbp := NewRunmountBuildpack()

	result, _ := rmbp.Detect(getContext())

	assert.True(t, !result.Compatible)
}
