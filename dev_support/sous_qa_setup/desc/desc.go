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

// LoadDesc loads an EnvDesc from a path.
func LoadDesc(descPath string) (EnvDesc, error) {
	var desc EnvDesc

	descReader, err := os.Open(descPath)
	if err != nil {
		return desc, err
	}

	dec := json.NewDecoder(descReader)
	err = dec.Decode(&desc)

	return desc, nil
}
