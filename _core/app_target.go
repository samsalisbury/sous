package core

import "github.com/opentable/sous/tools/docker"

type AppTarget struct {
	Context      *Context
	Buildpack    *RunnableBuildpack
	Command      string
	ArtifactPath string
}

func NewAppTarget(bp *RunnableBuildpack, c *Context) *AppTarget {
	return &AppTarget{c, bp, "-command set by compile target-",
		"-artifact set by compile target-"}
}

func (t *AppTarget) Name() string { return "app" }

func (t *AppTarget) DependsOn() []Target {
	return []Target{NewCompileTarget(t.Buildpack, t.Context)}
}

func (t *AppTarget) SetState(name string, value interface{}) {
	if name != "compile" {
		return
	}
	m, ok := value.(map[string]string)
	if !ok {
		panic("compile target returned a %T; want map[string]string")
	}
	artifactPath, ok := m["artifactPath"]
	if !ok {
		panic("compile target returned map with no artifactPath")
	}
	command, ok := m["command"]
	if !ok {
		panic("compile target returned map with no command")
	}
	t.ArtifactPath = artifactPath
	t.Command = command
}

func (t *AppTarget) String() string { return t.Name() }

func (t *AppTarget) Desc() string {
	return "generates artifacts for injection into a production container"
}

func (t *AppTarget) Check() error { return nil }

func (t *AppTarget) Dockerfile(c *TargetContext) *docker.File {
	image := c.Buildpack.StackVersion.GetBaseImageTag("app")
	df := &docker.File{From: image}
	df.Maintainer = c.User
	df.WORKDIR("/srv/app")
	df.ADD(".", t.ArtifactPath)
	df.CMD(t.Command)
	return df
}

// DockerRun returns a configured *docker.Run, which is used to create a new
// container when the old one is stale or does not exist.
func (t *AppTarget) DockerRun(tc *TargetContext) *docker.Run {
	r := docker.NewRun(tc.DockerTag())
	return r
}
