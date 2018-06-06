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

// func TestRunmountBuilder_Runmount(t *testing.T) {
// 	sh, _ := shell.Default()
// 	ctx := sous.BuildContext{
// 		Sh: sh,
// 	}

// 	buildContainerID, err := createMountContainer(ctx, testID)
// 	fmt.Println("err : ", err)
// 	fmt.Println("buildContainerID : ", buildContainerID)

// 	runSpec, err := extractRunSpec(ctx, tempDir, buildContainerID)
// 	fmt.Println("err : ", err)
// 	fmt.Println("runSpec : ", runSpec)

// 	err = validateRunSpec(runSpec)
// 	fmt.Println("err : ", err)

// 	subBuilders, err := constructImageBuilders(runSpec)
// 	fmt.Println("err : ", err)
// 	fmt.Println("subBuilders : ", *subBuilders[0])

// 	err = extractFiles(ctx, buildContainerID, tempDir, subBuilders)
// 	fmt.Println("err : ", err)

// 	err = teardownBuildContainer(ctx, buildContainerID)
// 	fmt.Println("err : ", err)

// 	err = templateDockerfile(ctx, tempDir, subBuilders)
// 	fmt.Println("templateDockerfile err : ", err)

// 	err = buildRunnables(ctx, tempDir, subBuilders)
// 	fmt.Println("err : ", err)

// 	products := products(ctx, subBuilders)
// 	fmt.Println("products : ", products)

// 	//	assert.FailNow(t, "")
// }
