package desc

import (
	"encoding/json"
	"net"
	"os"
)

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

// Complete returns false if any filed of the EnvDesc has been left empty.
// This is useful because e.g. as fields are added across branches, it's easy
// for tests to rely on data that was left unset by older code.
func (ed EnvDesc) Complete() bool {
	return ed.RegistryName != "" &&
		ed.SingularityURL != "" &&
		ed.GitOrigin != "" &&
		len(ed.AgentIP) > 0
}
