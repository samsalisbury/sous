package docker

import (
	"fmt"
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
	dockerTag := d.ImageTag(v)
	start := time.Now()
	err := c.Sh.Run("docker", "build", "-t", dockerTag, ".")
	if err != nil {
		return nil, err
	}
	return &sous.BuildResult{
		ImageName: dockerTag,
		Elapsed:   time.Since(start),
	}, nil
}

func (d *DockerfileBuildpack) ImageTag(v sous.SourceVersion) string {
	regRepo := filepath.Join(d.DockerRegistryHost, v.CanonicalName().String())
	return fmt.Sprintf("%s:%s", regRepo, v.Version)
}
