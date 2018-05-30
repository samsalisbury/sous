package docker

import (
	"fmt"
	"time"

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
		Compatible: true, //ctx.Source.DevBuild, // TODO LH only for testing porpoises
	}
	rmbp.detected = &result
	return &result, nil
}

func (rmbp *RunmountBuildpack) Build(ctx *sous.BuildContext) (*sous.BuildResult, error) {
	fmt.Println("Runmount Build.. ")
	start := time.Now()
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

	err = templateDockerfile(*ctx, tempDir, subBuilders)
	fmt.Println("templateDockerfile err : ", err)

	err = buildRunnables(*ctx, tempDir, subBuilders)
	fmt.Println("err : ", err)
	//fmt.Println("runnables : ", runnables)
	// TODO LH need these runnables to go to products.
	// TODO LH need to fix the naming of the runnables/subbuilders in here

	products := products(*ctx, subBuilders)
	fmt.Println("products : ", products)

	buildResult.Elapsed = time.Since(start)
	buildResult.Products = products

	return buildResult, nil
}
