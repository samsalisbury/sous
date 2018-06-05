package docker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
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

func readDockerfile() (string, error) {
	b, err := ioutil.ReadFile("Dockerfile")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (rmbp *RunmountBuildpack) Detect(ctx *sous.BuildContext) (*sous.DetectResult, error) {
	dfPath := filepath.Join(ctx.Source.OffsetDir, "Dockerfile")
	if !ctx.Sh.Exists(dfPath) {
		return nil, errors.New(fmt.Sprintf("%s does not exist", dfPath))
	}

	messages.ReportLogFieldsMessage("Runmount dockerfile detection", logging.DebugLevel, logging.Log, dfPath)

	dockerContent, _ := readDockerfile()

	// TODO LH simplest check so far, scan docker content for runmount
	result := sous.DetectResult{
		Compatible: strings.Contains(dockerContent, "runmount"),
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
