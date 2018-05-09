package docker

import (
	"fmt"

	sous "github.com/opentable/sous/lib"
)

type (
	RunmountBuildpack struct {
		detected *sous.DetectResult
	}
)

// const SOUS_RUN_IMAGE_SPEC = "SOUS_RUN_IMAGE_SPEC"

func NewRunmountBuildpack() *RunmountBuildpack {
	return &RunmountBuildpack{}
}

func (rmbp *RunmountBuildpack) Detect(ctx *sous.BuildContext) (*sous.DetectResult, error) {
	fmt.Println("Runmount Detector.. always return true")
	result := sous.DetectResult{
		Compatible: true,
	}
	rmbp.detected = &result
	return &result, nil
}

func (rmbp *RunmountBuildpack) Build(ctx *sous.BuildContext) (*sous.BuildResult, error) {
	fmt.Println("Runmount Build.. ")
	buildResult := &sous.BuildResult{}

	buildID, err := build(*ctx)
	fmt.Println("buildID :", buildID)

	err = run(*ctx, buildID)

	tempDir, err := setupTempDir()
	fmt.Println("tempDir : ", tempDir)

	buildContainerID, err := createMountContainer(*ctx, buildID)
	fmt.Println("buildContainerID : ", buildContainerID)

	runspec, err := extractRunSpec(*ctx, tempDir, buildContainerID)
	fmt.Println("runspec : ", runspec)

	err = validateRunSpec(runspec)
	fmt.Println("err : ", err)

	subBuilders, err := constructImageBuilders(runspec)
	fmt.Println("err : ", err)
	fmt.Println("subBuilders : ", *subBuilders[0])

	err = extractFiles(*ctx, buildContainerID, tempDir, subBuilders)
	fmt.Println("err : ", err)

	err = teardownBuildContainer(*ctx, buildContainerID)
	fmt.Println("err : ", err)

	return buildResult, nil
}
