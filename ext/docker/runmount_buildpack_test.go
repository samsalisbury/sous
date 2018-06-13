package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

// return the directory for cleanup
func createDockerfile(content string) string {
	dir, _ := ioutil.TempDir("", "sous-test")
	dockerfile := filepath.Join(dir, "Dockerfile")
	f, err := os.Create(dockerfile)
	if err != nil {
		fmt.Println("error creating file")
	}
	defer f.Close()
	f.WriteString(content)
	return dir
}

func removeDockerfile(dir string) {
	os.RemoveAll(dir)
}

func getContext(dir string) *sous.BuildContext {
	sh, _ := shell.Default()
	ctx := &sous.BuildContext{
		Sh: sh,
		Source: sous.SourceContext{
			OffsetDir: dir,
		},
	}
	return ctx
}

func TestRunmountBuildpack_DetectRunmount(t *testing.T) {
	dir := createDockerfile("FROM docker.otenv.com/sous-otj-autobuild-runmount:local")
	defer removeDockerfile(dir)

	rmbp := NewRunmountBuildpack(logging.SilentLogSet())

	result, _ := rmbp.Detect(getContext(dir))

	assert.True(t, result.Compatible)
}

func TestRunmountBuildpack_DetectNotRunmount(t *testing.T) {
	dir := createDockerfile("FROM docker.otenv.com/sous-otj-autobuild:local")
	defer removeDockerfile(dir)

	rmbp := NewRunmountBuildpack(logging.SilentLogSet())

	result, _ := rmbp.Detect(getContext(dir))

	assert.True(t, !result.Compatible)
}
