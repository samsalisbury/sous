package docker

import (
	"fmt"

	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/cmd"
)

type Container interface {
	CID() string
	Name() string
	Image() string
	String() string
	Kill() error
	KillIfRunning() error
	Stop(signal string) error
	Remove() error
	ForceRemove() error
	Start() error
	Exists() bool
	Running() bool
}

type container struct {
	cid, name string
}

func NewContainer(name, id string) Container {
	return &container{id, name}
}

func ContainerWithName(name string) Container {
	return &container{"", name}
}

func ContainerWithCID(cid string) Container {
	return &container{cid, ""}
}

func (c *container) CID() string  { return c.cid }
func (c *container) Name() string { return c.name }

func (c *container) Inspect() (*DockerContainer, bool) {
	if cmd.ExitCode("docker", "inspect", c.effectiveName()) != 0 {
		return nil, false
	}
	var dc []*DockerContainer
	cmd.JSON(&dc, "docker", "inspect", c.effectiveName())
	if len(dc) == 0 {
		return nil, false
	}
	if len(dc) != 1 {
		cli.Fatalf("Docker inspect %s returned more than one result, please open a GitHub issue about this",
			c.effectiveName())
	}
	// The second return value checks that the thing we inspected was, in fact, the
	// relevant container, since docker inpect takes both container and image
	// names and IDs, so it's a bit ambiguous otherwise.
	cont := dc[0]
	return cont, cont.Name == "/"+c.Name() || cont.ID == c.CID()
}

func (c *container) Exists() bool {
	_, exists := c.Inspect()
	return exists
}

func (c *container) Running() bool {
	dc, ok := c.Inspect()
	if !ok {
		return false
	}
	return dc.State.Running
}

func (c *container) Image() string {
	var dc []DockerContainer
	cmd.JSON(&dc, "docker", "inspect", c.Name())
	if len(dc) == 0 {
		cli.Fatalf("Container %s does not exist.", c)
	}
	if len(dc) != 1 {
		cli.Fatalf("Multiple containers match %s", c)
	}
	return dc[0].Image
}

type DockerContainer struct {
	ID, Name, Image string
	State           struct {
		Running, Paused, Restarting bool
	}
}

func (c *container) KillIfRunning() error {
	if err := c.Kill(); err != nil {
		if c.Running() {
			return err
		}
	}
	return nil
}

func (c *container) Stop(signal string) error {
	if ex := cmd.ExitCode("docker", "stop", "-s", signal, c.effectiveName()); ex != 0 {
		return fmt.Errorf("Unable to send %s signal to docker container %s", signal, c)
	}
	return nil
}

func (c *container) Kill() error {
	if ex := cmd.ExitCode("docker", "kill", c.effectiveName()); ex != 0 {
		return fmt.Errorf("Unable to kill docker container %s", c)
	}
	return nil
}

func (c *container) Remove() error {
	if ex := cmd.ExitCode("docker", "rm", c.effectiveName()); ex != 0 {
		return fmt.Errorf("Unable to remove docker container %s", c)
	}
	return nil
}

func (c *container) ForceRemove() error {
	if ex := cmd.ExitCode("docker", "rm", "-f", c.effectiveName()); ex != 0 {
		return fmt.Errorf("Unable to remove docker container %s", c)
	}
	return nil
}

func (c *container) Start() error {
	if ex := cmd.ExitCode("docker", "start", c.effectiveName()); ex != 0 {
		return fmt.Errorf("Unable to start docker container %s", c)
	}
	return nil
}

func (c *container) Wait() error {
	if ex := cmd.ExitCode("docker", "wait", c.effectiveName()); ex != 0 {
		return fmt.Errorf("Unable to wait on docker container %s", c)
	}
	return nil
}

func (c *container) String() string {
	return c.effectiveName()
}

func (c *container) effectiveName() string {
	if c.cid == "" {
		return c.name
	}
	return c.cid
}
