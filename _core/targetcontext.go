package core

import (
	"github.com/opentable/sous/tools/docker"
)

type TargetContext struct {
	*Context
	Target
	Buildpack  *RunnableBuildpack
	BuildState *BuildState
	TargetName string
}

// DockerTag returns the docker tag used for the current build.
func (c *TargetContext) DockerTag() string {
	return c.DockerTagForBuildNumber(c.BuildNumber())
}

// Dockerfile return the dockerfile used for this current build.
func (tc *TargetContext) Dockerfile() *docker.File {
	return tc.Target.Dockerfile(tc)
}

// BuildNumber returns the build number for the current project at its
// present commit on this machine with this user login. Heh, a mouthful.
func (c *TargetContext) BuildNumber() int {
	return c.BuildState.CurrentCommit().BuildNumber
}

// PrevDockerTag returns the previously built docker tag for this project.
// This is useful for re-using builds when appropriate.
func (c *TargetContext) PrevDockerTag() string {
	return c.DockerTagForBuildNumber(c.BuildNumber() - 1)
}
