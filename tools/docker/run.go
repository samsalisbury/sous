package docker

import (
	"fmt"
	"os/exec"

	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/cmd"
	"github.com/opentable/sous/tools/file"
)

type Run struct {
	Image, Name            string
	UserID                 string
	ReRun                  Container
	Env                    map[string]string
	Net                    string
	StdoutFile, StderrFile string
	Volumes                []string
	// Args are args passed to the container (as opposed to the
	// docker run command).
	Args []string
	// Command is a command passed to the container before Args if
	// it is specified.
	Command      string
	Labels       map[string]string
	inBackground bool
}

func NewRun(imageTag string) *Run {
	return &Run{
		Image: imageTag,
		Net:   "host",
		Env:   map[string]string{},
		// This will not be necessary when running docker 1.9+, see https://github.com/docker/docker/issues/17964
		Labels: getLabelsFromImage(imageTag),
	}
}

func getLabelsFromImage(imageTag string) map[string]string {
	if !ImageExists(imageTag) {
		cli.Fatalf("cannot find image %s", imageTag)
	}
	if !ExactlyOneImageExists(imageTag) {
		cli.Fatalf("multiple images named %s, cannot continue", imageTag)
	}
	var images []*Image
	cmd.JSON(&images, "docker", "inspect", imageTag)
	image := images[0]
	return image.Config.Labels
}

func NewReRun(container Container) *Run {
	return &Run{
		ReRun: container,
	}
}

func (r *Run) AddEnv(key, value string) {
	r.Env[key] = value
}

func (r *Run) AddLabel(key, value string) {
	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	r.Labels[key] = value
}

func (r *Run) AddLabels(labels map[string]string) {
	for k, v := range labels {
		r.AddLabel(k, v)
	}
}

func (r *Run) AddVolume(hostPath, containerPath string) {
	if r.Volumes == nil {
		r.Volumes = []string{}
	}
	r.Volumes = append(r.Volumes, fmt.Sprintf("%s:%s", hostPath, containerPath))
}

func (r *Run) Background() *Run {
	r.inBackground = true
	return r
}

func (r *Run) prepareCommand() *cmd.CMD {
	var args []string
	if r.ReRun != nil {
		// Add -i flag since start by default puts container in background
		args = []string{"start", "-i", r.ReRun.String()}
	} else {
		args = []string{"run"}
		if r.inBackground {
			args = append(args, "-d")
		}
		if r.Name != "" {
			args = append(args, "--name", r.Name)
		}
		for k, v := range r.Env {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}
		for _, v := range r.Volumes {
			args = append(args, "-v", v)
		}
		if r.Net != "" {
			args = append(args, "--net="+r.Net)
		}
		if r.UserID != "" {
			args = append(args, "-u", r.UserID)
		}
		for k, v := range r.Labels {
			args = append(args, "-l", fmt.Sprintf("%s=%s", k, v))
		}
		// Do not add more options after this line.
		args = append(args, r.Image)
		if r.Command != "" {
			args = append(args, r.Command)
		}
		args = append(args, r.Args...)
	}
	c := dockerCmd(args...)
	if r.inBackground {
		c.EchoStdout = false
		c.EchoStderr = false
	}
	return c
}

func (r *Run) ExitCode() int {
	return r.prepareCommand().ExitCode()
}

func (r *Run) CalculatedCommand() string {
	return r.prepareCommand().String()
}

func (r *Run) Start() (*container, error) {
	r.inBackground = true
	c := r.prepareCommand()
	cid := c.Out()
	tailLogs := exec.Command("docker", "logs", "-f", cid)
	if r.StdoutFile == "" {
		cli.Fatalf("You must set docker.Run.StdoutFile before calling .Start()")
	}
	if r.StderrFile == "" {
		cli.Fatalf("You must set docker.Run.Stderr before calling .Start()")
	}
	tailLogs.Stdout = file.Create(r.StdoutFile)
	tailLogs.Stderr = file.Create(r.StderrFile)
	err := tailLogs.Start()
	if err != nil {
		cli.Fatalf("Unable to tail logs: %s", err)
	}
	return &container{cid, ""}, nil
}
