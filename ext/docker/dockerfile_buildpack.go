package docker

import (
	"fmt"
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
	if !c.Sh.Exists("Dockerfile") {
		return nil, fmt.Errorf("Dockerfile does not exist")
	}

	start := time.Now()
	output, err := c.Sh.Stdout("docker", "build", ".")
	if err != nil {
		return nil, err
	}

	match := successfulBuildRE.FindStringSubmatch(string(output))
	if match == nil {
		return nil, fmt.Errorf("Couldn't find container id in:\n%s", output)
	}

	return &sous.BuildResult{
		ImageID: match[1],
		Elapsed: time.Since(start),
	}, nil
}

func (d *DockerfileBuildpack) Detect(c *sous.BuildContext) (*sous.DetectResult, error) {
	if !c.Sh.Exists("Dockerfile") {
		return nil, fmt.Errorf("Dockerfile does not exist")
	}
	return &sous.DetectResult{Compatible: true}, nil
}

/*
func buildAndPushContainer(containerDir, tagName string) error {
	build := exec.Command("docker", "build", ".")
	build.Dir = containerDir
	output, err := build.CombinedOutput()
	if err != nil {
		log.Print("Problem building container: ", containerDir, "\n", string(output))
		return err
	}

	match := successfulBuildRE.FindStringSubmatch(string(output))
	if match == nil {
		return fmt.Errorf("Couldn't find container id in:\n%s", output)
	}

	cID := match[1]
	tag := exec.Command("docker", "tag", cID, tagName)
	tag.Dir = containerDir
	output, err = tag.CombinedOutput()
	if err != nil {
		return err
	}

	push := exec.Command("docker", "push", tagName)
	push.Dir = containerDir
	output, err = push.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}
*/
