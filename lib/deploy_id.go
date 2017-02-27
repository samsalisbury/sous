package sous

import (
	"fmt"
	"strings"
)

// A DeployID identifies a deployment.
type DeployID struct {
	ManifestID ManifestID
	Cluster    string
}

// ParseDeployID parses a deployID in the format: ManifestID:Cluster.
func ParseDeployID(s string) (DeployID, error) {
	lastColonIndex := strings.LastIndex(s, ":")
	if lastColonIndex == -1 {
		return DeployID{}, fmt.Errorf("invalid deployID string %q; should have a colon separating manifest ID from cluster", s)
	}
	mid := s[:lastColonIndex]
	cluster := s[lastColonIndex+1:]
	if len(mid) == 0 {
		return DeployID{}, fmt.Errorf("invalid deployID string %q; should begin with source location", s)
	}
	if len(cluster) == 0 {
		return DeployID{}, fmt.Errorf("invalid deployID string %q; empty cluster section", s)
	}
	manifestID, err := ParseManifestID(mid)
	if err != nil {
		return DeployID{}, err
	}
	return DeployID{
		ManifestID: manifestID,
		Cluster:    cluster,
	}, nil
}

// MustParseDeployID wraps ParseDeployID and panics if it returns an error,
// otherwise returns the DeployID returned.
func MustParseDeployID(s string) DeployID {
	did, err := ParseDeployID(s)
	if err != nil {
		panic(err)
	}
	return did
}

func (did DeployID) String() string {
	return fmt.Sprintf("%s:%s", did.ManifestID, did.Cluster)
}
