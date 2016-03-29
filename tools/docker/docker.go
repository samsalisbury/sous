package docker

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/cmd"
	"github.com/opentable/sous/tools/dir"
	"github.com/opentable/sous/tools/dockermachine"
	"github.com/opentable/sous/tools/file"
	"github.com/opentable/sous/tools/version"
)

func RequireVersion(r *version.R) {
	vs := cmd.Table("docker", "--version")[0][2]
	vs = strings.Trim(vs, ",")
	v := version.Version(vs)
	if !r.IsSatisfiedBy(v) {
		cli.Fatalf("got docker version %s; want %s", v, r)
	}
}

func GetDockerHost() string {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		return cmd.Stdout("hostname")
	}
	u, err := url.Parse(dockerHost)
	if err != nil {
		return cmd.Stdout("hostname")
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return cmd.Stdout("hostname")
	}
	return host
}

func RequireDaemon() {
	if c := cmd.ExitCode("docker", "ps"); c != 0 {
		cli.Logf("`docker ps` exited with code %d", c)
		if dockermachine.Installed() {
			vms := dockermachine.RunningVMs()
			if len(vms) != 0 {
				cli.Fatalf(`Tip: eval "$(docker-machine env %s)"`, vms[0])
			}
			vms = dockermachine.VMs()
			switch len(vms) {
			case 0:
				cli.Logf("Tip: you should create a machine using docker-machine")
			case 1:
				start := ""
				if cmd.Stdout("docker-machine", "status", vms[0]) != "Running" {
					start = fmt.Sprintf("docker-machine start %s && ", vms[0])
				}
				cli.Logf(`Tip: %seval "$(docker-machine env %s)"`, start, vms[0])
			default:
				cli.Logf("Tip: start one of your docker machines (%s)",
					strings.Join(vms, ", "))
			}
		}
		cli.Fatal()
	}
}

func dockerCmd(args ...string) *cmd.CMD {
	c := cmd.New("docker", args...)
	c.EchoStdout = true
	c.EchoStderr = true
	return c
}

// Build builds the dockerfile in the specified directory and returns the image ID
func Build(directory, tag string) string {
	return dockerCmd("build", "-t", tag, dir.Resolve(directory)).Out()
}

// BuildFile builds the specified docker file in the context of the specified
// directory.
func BuildFile(dockerfile, directory, tag string) string {
	if !file.Exists(dockerfile) {
		cli.Fatalf("File does not exist: %s", dockerfile)
	}
	directory = dir.Resolve(directory)
	localDockerfile := ".SousDockerfile"
	if file.Exists(localDockerfile) {
		file.Remove(localDockerfile)
	}
	file.RemoveOnExit(localDockerfile)
	// If there is a .gitignore, but no .dockerignore, link it as .dockerignore
	if file.Exists(".gitignore") {
		if file.Exists(".dockerignore") {
			cli.Warn("./.dockerignore found, it is recommended to remove this so Sous can use your .gitignore")
		} else {
			file.TemporaryLink(".gitignore", ".dockerignore")
			// We try to clean this file up early, in preperation for the next build step
			defer file.Remove(".dockerignore")
		}
	}
	file.TemporaryLink(dockerfile, localDockerfile)
	// We try to clean the local Dockerfile up early, in preperation for the next build step
	defer file.Remove(localDockerfile)
	return dockerCmd("build", "-f", localDockerfile, "-t", tag, directory).Out()
}

func Push(tag string) {
	cmd.EchoAll("docker", "push", tag)
}

func ImagesMatchingTag(tag string) []string {
	return cmd.Lines("docker", "images", "--no-trunc", "-q", tag)
}

func ImageExists(tag string) bool {
	return len(ImagesMatchingTag(tag)) != 0
}

func ExactlyOneImageExists(tag string) bool {
	return len(ImagesMatchingTag(tag)) == 1
}

func Pull(image string) string {
	cmd.EchoAll("docker", "pull", image)
	return ImageID(image)
}

func Layers(image string) []string {
	t := cmd.Table("docker", "history", "--no-trunc", image)
	layers := make([]string, len(t))
	for i, r := range t {
		layers[i] = r[0]
	}
	return layers
}

func ImageID(image string) string {
	ids := ImagesMatchingTag(image)
	if len(ids) == 0 || ids[0] == "" {
		cli.Fatalf("image missing: %s", image)
	}
	if len(ids) != 1 {
		cli.Fatalf("multiple images match %s; sous requires exact tags", image)
	}
	return ids[0]
}

func Commit(cid, tag string) error {
	if code := cmd.ExitCode("docker", "commit", cid, tag); code != 0 {
		return fmt.Errorf("Command failed with exit code %d", code)
	}
	return nil
}

func BaseImageUpdated(baseImageTag, builtImageTag string) bool {
	if !ImageExists(baseImageTag) {
		return true
	}
	baseImageID := ImageID(baseImageTag)
	layers := Layers(builtImageTag)
	for _, l := range layers {
		if l == baseImageID {
			return false
		}
	}
	return true
}
