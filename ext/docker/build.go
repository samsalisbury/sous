package docker

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/opentable/sous/sous"
)

type (
	// DockerfileBuildpack is a simple buildpack for building projects using
	// their own Dockerfile.
	DockerfileBuildpack struct {
		DockerRegistryHost string
	}
)

// NewDockerfileBuildpack creates a Dockerfile buildpack
func NewDockerfileBuildpack(registry string) *DockerfileBuildpack {
	return &DockerfileBuildpack{
		DockerRegistryHost: registry,
	}
}

// Build implements sous.Buildpack.Build
func (d *DockerfileBuildpack) Build(c *sous.BuildContext) (*sous.BuildResult, error) {
	if !c.Sh.Exists("Dockerfile") {
		return nil, fmt.Errorf("Dockerfile does not exist")
	}
	v := c.Source.Version()
	start := time.Now()
	err := c.Sh.Run("docker", "build", ".")
	if err != nil {
		return nil, err
	}
	return &sous.BuildResult{
		ImageName: dockerTag,
		Elapsed:   time.Since(start),
	}, nil
}

// ImageTag computes an image tag from a SourceVersion
func (d *DockerfileBuildpack) ImageTag(v sous.SourceVersion) string {
	regRepo := filepath.Join(d.DockerRegistryHost, v.CanonicalName().String())
	return fmt.Sprintf("%s:%s", regRepo, v.Version)
}

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
