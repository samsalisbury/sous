package desc

import "net"

type (
	// EnvDesc captures the details of the established environment
	EnvDesc struct {
		RegistryName   string
		SingularityURL string
		GitOrigin      string
		AgentIP        net.IP
	}
)
