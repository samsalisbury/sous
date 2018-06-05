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

func TestRunmountBuilder_Runmount(t *testing.T) {
	sh, _ := shell.Default()
	ctx := sous.BuildContext{
		Sh: sh,
	}
	buildID := "3d9fac8b3558"
	// err := run(ctx, buildID)
	// if err != nil {
	// 	fmt.Println("err : ", err)
	// }

	//artifacts should be built now grab them
	tempDir, err := setupTempDir()
	fmt.Println("err : ", err)
	fmt.Println("tempDir : ", tempDir)

	buildContainerID, err := createMountContainer(ctx, buildID)
	fmt.Println("err : ", err)
	fmt.Println("buildContainerID : ", buildContainerID)

	runSpec, err := extractRunSpec(ctx, tempDir, buildContainerID)
	fmt.Println("err : ", err)
	fmt.Println("runSpec : ", runSpec)

	err = validateRunSpec(runSpec)
	fmt.Println("err : ", err)

	subBuilders, err := constructImageBuilders(runSpec)
	fmt.Println("err : ", err)
	fmt.Println("subBuilders : ", *subBuilders[0])

	err = extractFiles(ctx, buildContainerID, tempDir, subBuilders)
	fmt.Println("err : ", err)

	err = teardownBuildContainer(ctx, buildContainerID)
	fmt.Println("err : ", err)

	err = templateDockerfile(ctx, tempDir, subBuilders)
	fmt.Println("templateDockerfile err : ", err)

	err = buildRunnables(ctx, tempDir, subBuilders)
	fmt.Println("err : ", err)

	products := products(ctx, subBuilders)
	fmt.Println("products : ", products)

	assert.FailNow(t, "")
}
