package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

var testBuildID = "cabba9edeadbeef"
var testContainerBuildID = "deadbeef"
var testRunID = "A90110"

func TestRunmountBuilder_build(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "build")
	cctl.ResultSuccess(fmt.Sprintf("Successfully built %s", testBuildID), "")
	ctx := sous.BuildContext{
		Sh: sh,
	}
	buildID, _ := build(ctx)
	assert.Equal(t, testBuildID, buildID)
}

// This isn't testing much other than docker run exited with a 0
func TestRunmountBuild_run(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "run")
	cctl.ResultSuccess("finished", "")
	ctx := sous.BuildContext{
		Sh: sh,
	}

	err := run(ctx, testBuildID)
	assert.Empty(t, err)
}

func TestRunmountBuild_setupTempDir(t *testing.T) {
	dirCreated, err := setupTempDir()
	assert.Empty(t, err)
	_, err = os.Stat(dirCreated)
	assert.True(t, err == nil)
}

func TestRunmountBuild_createMountContainer(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "create")
	cctl.ResultSuccess(fmt.Sprintf("%s", testContainerBuildID), "")
	ctx := sous.BuildContext{
		Sh: sh,
	}

	containerID, err := createMountContainer(ctx, testBuildID)
	assert.Empty(t, err)
	assert.Equal(t, testContainerBuildID, containerID)
}

func getTestRunSpec() MultiImageRunSpec {
	testRunSpec := MultiImageRunSpec{}
	specF, _ := os.Open("./testdata/runmountbuilder/run_spec.json")
	dec := json.NewDecoder(specF)
	dec.Decode(&testRunSpec)

	return testRunSpec
}

func TestRunmountBuild_extractRunspec(t *testing.T) {
	tempDir := "./testdata/runmountbuilder"
	testRunSpec := getTestRunSpec()
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "cp")
	cctl.ResultSuccess("", "")
	ctx := sous.BuildContext{
		Sh: sh,
	}

	runSpec, _ := extractRunSpec(ctx, tempDir, testContainerBuildID)
	fmt.Println("runspec : ", runSpec)

	assert.Equal(t, testRunSpec, runSpec)
}

func TestRunmountBuild_validateRunSpec(t *testing.T) {
	testRunSpec := getTestRunSpec()
	err := validateRunSpec(testRunSpec)
	assert.NoError(t, err)
}

func TestRunmountBuild_constructImageBuilders(t *testing.T) {
	testRunSpec := getTestRunSpec()
	builders, err := constructImageBuilders(testRunSpec)

	assert.NoError(t, err)
	builder := *builders[0]
	assert.Equal(t, "docker.otenv.com/sous-otj-run", builder.RunSpec.Image.From)
}

func TestRunmountBuild_extractFiles(t *testing.T) {
	testRunSpec := getTestRunSpec()
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "cp")
	cctl.ResultSuccess("", "")
	ctx := sous.BuildContext{
		Sh: sh,
	}
	builders, _ := constructImageBuilders(testRunSpec)
	err := extractFiles(ctx, testContainerBuildID, "/tmp", builders)
	assert.NoError(t, err)
}

func TestRunmountBuild_templateDockerfile(t *testing.T) {
	testRunSpec := getTestRunSpec()
	buildDir := "/tmp"
	builders, _ := constructImageBuilders(testRunSpec) //could abstract this for testing

	err := templateDockerfile(sous.BuildContext{}, buildDir, builders)
	assert.NoError(t, err)
}

func TestRunmountBuild_buildRunnables(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "build")
	cctl.ResultSuccess(fmt.Sprintf("Successfully built %s", testRunID), "")
	ctx := sous.BuildContext{
		Sh: sh,
	}
	builders, _ := constructImageBuilders(getTestRunSpec())
	err := buildRunnables(ctx, "/tmp", builders)
	fmt.Println("err : ", err)
	assert.NoError(t, err)
}

func TestRunmountBuild_products(t *testing.T) {
	builders, _ := constructImageBuilders(getTestRunSpec())
	products := products(sous.BuildContext{}, builders)
	assert.Equal(t, 1, len(products))
}
