package desc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type (
	service struct {
		Host net.IP
		Port uint
	}

	// EnvDesc captures the details of the established environment
	EnvDesc struct {
		DockerRegistry, Git, Singularity service
		AgentIP                          net.IP
	}
)

// SingularityURL returns a URL for the test Singularity service.
func (ed EnvDesc) SingularityURL() string {
	return fmt.Sprintf("http://%s/singularity", ed.Singularity.hostPort())
}

// GitOrigin returns the ip:port for the test Git instance.
func (ed EnvDesc) GitOrigin() string {
	return ed.Git.hostPort()
}

// RegistryName returns the ip:port for the test Docker registry.
func (ed EnvDesc) RegistryName() string {
	return ed.DockerRegistry.hostPort()
}

func (serv service) hostPort() string {
	return fmt.Sprintf("%s:%d", serv.Host, serv.Port)
}

func (serv service) complete() bool {
	return len(serv.Host) > 0 &&
		serv.Port != 0
}

// LoadDesc loads an EnvDesc from a path.
func LoadDesc(descPath string) (EnvDesc, error) {
	var desc EnvDesc

	descReader, err := os.Open(descPath)
	if err != nil {
		return desc, err
	}

	dec := json.NewDecoder(descReader)
	err = dec.Decode(&desc)

	return desc, err
}

// Complete returns false if any filed of the EnvDesc has been left empty.
// This is useful because e.g. as fields are added across branches, it's easy
// for tests to rely on data that was left unset by older code.
func (ed EnvDesc) Complete() bool {
	return ed.DockerRegistry.complete() &&
		ed.Singularity.complete() &&
		ed.Git.complete() &&
		len(ed.AgentIP) > 0
}
