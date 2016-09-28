package docker

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/opentable/sous/lib"
)

type (
	// DockerfileBuildpack is a simple buildpack for building projects using
	// their own Dockerfile.
	DockerfileBuildpack struct {
	}
)

// NewDockerfileBuildpack creates a Dockerfile buildpack
func NewDockerfileBuildpack() *DockerfileBuildpack {
	return &DockerfileBuildpack{}
}

var successfulBuildRE = regexp.MustCompile(`Successfully built (\w+)`)

// Build implements Buildpack.Build
func (d *DockerfileBuildpack) Build(c *sous.BuildContext) (*sous.BuildResult, error) {
	start := time.Now()
	offset := "."
	if c.Source.OffsetDir != "" {
		offset = c.Source.OffsetDir
	}
	output, err := c.Sh.Stdout("docker", "build", offset)
	if err != nil {
		return nil, err
	}

	match := successfulBuildRE.FindStringSubmatch(string(output))
	if match == nil {
		return nil, fmt.Errorf("Couldn't find container id in:\n%s", output)
	}

	return &sous.BuildResult{
		ImageID:    match[1],
		Elapsed:    time.Since(start),
		Advisories: c.Advisories,
	}, nil
}

func (d *DockerfileBuildpack) Detect(c *sous.BuildContext) (*sous.DetectResult, error) {
	if !c.Sh.Exists(filepath.Join(c.Source.OffsetDir, "Dockerfile")) {
		return nil, fmt.Errorf("Dockerfile does not exist")
	}
	return &sous.DetectResult{Compatible: true}, nil
}
