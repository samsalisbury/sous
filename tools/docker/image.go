package docker

import (
	"fmt"

	"github.com/opentable/sous/tools/cmd"
)

type Image struct {
	ID, Name, Tag string
	Config        *ImageConfig
}

type ImageConfig struct {
	Labels map[string]string
}

func NewImage(name, tag string) *Image {
	return &Image{Name: name, Tag: tag}
}

func (i *Image) Remove() error {
	exitCode := cmd.ExitCode("docker", "rmi", fmt.Sprintf("%s:%s", i.Name, i.Tag))
	if exitCode == 0 {
		return nil
	}
	return fmt.Errorf("exit code %d", exitCode)
}
