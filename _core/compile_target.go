package core

import (
	"fmt"
	"os/user"

	"github.com/opentable/sous/core/resources"
	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/dir"
	"github.com/opentable/sous/tools/docker"
	"github.com/opentable/sous/tools/file"
)

type CompileTarget struct {
	Context      *Context
	Buildpack    *RunnableBuildpack
	ArtifactPath string
}

func NewCompileTarget(bp *RunnableBuildpack, c *Context) *CompileTarget {
	return &CompileTarget{c, bp, ""}
}

func (t *CompileTarget) Name() string { return "compile" }

func (t *CompileTarget) DependsOn() []Target { return nil }

func (t *CompileTarget) State() interface{} {
	command, err := t.Buildpack.RunScript("command.sh",
		t.Buildpack.Scripts.Command, t.Context.WorkDir)
	if err != nil {
		cli.Fatal(err)
	}
	return map[string]string{
		"command":      command,
		"artifactPath": t.ArtifactPath,
	}
}

func (t *CompileTarget) String() string { return t.Name() }

func (t *CompileTarget) Desc() string {
	return "generates artifacts for injection into a production container"
}

func (t *CompileTarget) Check() error { return nil }

func (t *CompileTarget) Dockerfile(c *TargetContext) *docker.File {
	image := c.Buildpack.StackVersion.GetBaseImageTag(t.Name())
	df := &docker.File{From: image}
	// This is a non-portable container, since it includes the UID of the
	// logged-in user. This is necessary to ensure the user in the container
	// can write files accessible to the user invoking the container on the
	// host.
	u, err := user.Current()
	if err != nil {
		cli.Fatalf("unable to get current user: %s", err)
	}
	// Just use the username for group name, it doesn't matter as long as
	// the IDs are right.
	df.RUN("groupadd", "-g", u.Gid, u.Username)
	// Explanation of some of the below useradd flags:
	//   -M means do not create home directory, which we do not need
	//   --no-log-init means do not create a 32G sparse file (which Docker commit
	//       cannot handle properly, and tries to create a non-sparse 32G file.)
	df.RUN("useradd", "--no-log-init", "-M", "--uid", u.Uid, "--gid", u.Gid, u.Username)

	compileScript := resources.Files["build-prep.bash"] + "\n"
	compileScript += t.Buildpack.PrepareScript("compile.sh", t.Buildpack.Scripts.Compile) + "\n"

	compileScript += fmt.Sprintf("chown %s -R /artifacts", u.Name)
	file.RemoveOnExit("compile.sh")
	file.WriteString(compileScript, "compile.sh")
	df.ADD("/scripts/", "compile.sh")
	df.ENV(map[string]string{
		"ARTIFACT_DIR": "/mnt/artifacts",
	})
	df.RUN("mkdir", "/build")
	df.RUN("chmod", "777", "/scripts/*", "/build")
	return df
}

// DockerRun returns a configured *docker.Run, which is used to create a new
// container when the old one is stale or does not exist.
func (t *CompileTarget) DockerRun(tc *TargetContext) *docker.Run {
	r := docker.NewRun(tc.DockerTag())
	env := tc.Context.BuildpackEnv()
	for k, v := range env {
		r.AddEnv(k, v)
	}
	artifactDir := GetEmptyArtifactDir(tc)
	r.AddVolume(artifactDir, "/mnt/artifacts")
	r.AddVolume(tc.Git.Dir, "/mnt/repo")

	// This command makes an isolated pristine snapshot of the working tree
	// and then invovkes the compile.sh from the buildpack.
	r.Command = "/scripts/compile.sh"

	return r
}

func YESorNO(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

func GetEmptyArtifactDir(tc *TargetContext) string {
	artifactDir := tc.FilePath("artifacts")
	if dir.Exists(artifactDir) {
		dir.Remove(artifactDir)
	}
	dir.EnsureExists(artifactDir)
	return artifactDir
}
