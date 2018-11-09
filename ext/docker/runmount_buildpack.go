package docker

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/pkg/errors"
)

// BuildOutEnv is the other environment variable a runmount image must set
const BuildOutEnv = "BUILD_OUT"

// BuildCacheEnv is an environment variable a runmount image may set
const BuildCacheEnv = "BUILD_CACHE"

type (
	// RunmountBuildpack builds a container, runs it seperately to use docker mounts for cache and output,
	// and builds final deploy container
	RunmountBuildpack struct {
		registry docker_registry.Client
		detected *sous.DetectResult
		log      logging.LogSink
	}
)

// NewRunmountBuildpack returns a new RunmountBuildpack
func NewRunmountBuildpack(r docker_registry.Client, ls logging.LogSink) *RunmountBuildpack {
	return &RunmountBuildpack{
		registry: r,
		log:      ls,
	}
}

func readDockerfile(dfPath string) (string, error) {
	b, err := ioutil.ReadFile(dfPath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Detect implements Buildpack on SplitBuildpack
func (rmbp *RunmountBuildpack) Detect(ctx *sous.BuildContext) (*sous.DetectResult, error) {
	dfPath := filepath.Join(ctx.Source.OffsetDir, "Dockerfile")
	if !ctx.Sh.Exists(dfPath) {
		return nil, errors.Errorf("%s does not exist", dfPath)
	}
	messages.ReportLogFieldsMessage("Runmount dockerfile detection", logging.DebugLevel, rmbp.log, dfPath)

	detector, err := inspectDockerfile(ctx.Sh.Abs(dfPath), ctx.Source.DevBuild, ctx.Sh, dfPath, rmbp.registry, rmbp.log)

	rmbp.detected = &sous.DetectResult{
		Compatible: false,
	}
	if err == nil {
		specPath, hasRunspec := detector.envValue(SOUS_RUN_IMAGE_SPEC)
		buildOut, hasOut := detector.envValue(BuildOutEnv)
		buildCache, hasCache := detector.envValue(BuildCacheEnv)
		if hasRunspec && hasOut {
			data := detectData{
				RunImageSpecPath:  specPath,
				BuildOutPath:      buildOut,
				HasAppVersionArg:  detector.versionArg,
				HasAppRevisionArg: detector.revisionArg,
			}

			if hasCache {
				data.BuildCachePath = buildCache
			}

			rmbp.detected = &sous.DetectResult{Compatible: true, Data: data}
		}
	}

	return rmbp.detected, err
}

// Build implements Buildpack on RunmountBuildpack
func (rmbp *RunmountBuildpack) Build(ctx *sous.BuildContext) (*sous.BuildResult, error) {
	fmt.Println("Runmount Build.. ")
	start := time.Now()
	buildResult := &sous.BuildResult{}
	detection := rmbp.detected.Data.(detectData)

	buildID, err := build(*ctx)
	if err != nil {
		return nil, err
	}

	err = run(*ctx, detection, buildID)
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

	runspec, err := extractRunSpec(*ctx, detection, tempDir, buildContainerID)
	if err != nil {
		return nil, err
	}

	err = validateRunSpec(runspec)
	if err != nil {
		return nil, err
	}

	imageBuilders := constructImageBuilders(runspec)

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
