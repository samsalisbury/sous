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
	if err != nil {
		return nil, err
	}

	err = run(*ctx, buildID)
	if err != nil {
		return nil, err
	}

	tempDir, err := setupTempDir()
	if err != nil {
		return nil, err
	}

	buildContainerID, err := createMountContainer(*ctx, buildID)
	if err != nil {
		return nil, err
	}

	runspec, err := extractRunSpec(*ctx, tempDir, buildContainerID)
	if err != nil {
		return nil, err
	}

	err = validateRunSpec(runspec)
	if err != nil {
		return nil, err
	}

	imageBuilders, err := constructImageBuilders(runspec)
	if err != nil {
		return nil, err
	}

	err = extractFiles(*ctx, buildContainerID, tempDir, imageBuilders)
	if err != nil {
		return nil, err
	}

	err = teardownBuildContainer(*ctx, buildContainerID)
	if err != nil {
		return nil, err
	}

	err = templateDockerfile(*ctx, tempDir, imageBuilders)
	if err != nil {
		return nil, err
	}

	err = buildRunnables(*ctx, tempDir, imageBuilders)
	if err != nil {
		return nil, err
	}

	products := products(*ctx, imageBuilders)
	if err != nil {
		return nil, err
	}

	buildResult.Elapsed = time.Since(start)
	buildResult.Products = products

	return buildResult, nil
}
