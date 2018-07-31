package otpl

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/jsonutil"
)

type (
	// SingularityJSON represents the JSON in an otpl-deploy singularity.json
	// file. Note that the json tags are essential to validating parsed JSON
	// contains only recognised fields.
	SingularityJSON struct {
		RequestID string               `json:"requestId"`
		Resources SingularityResources `json:"resources"`
		Env       sous.Env             `json:"env,omitempty"`
	}
	// SingularityResources represents the resources section in SingularityJSON.
	SingularityResources map[string]float64
	// SingularityRequestJSON represents JSON in an otpl-deploy
	// singularity-request.json file.
	SingularityRequestJSON struct {
		ID          string `json:"id"`
		RequestType string `json:"requestType"`
		// Instances is the number of instances in this deployment.
		Instances int `json:"instances"`
		// Owners is a comma-separated list of email addresses.
		Owners []string `json:"owners"`
	}
)

func parseSingularityJSON(rawJSON string) (SingularityJSON, error) {
	v := SingularityJSON{}
	if err := jsonutil.StrictParseJSON(rawJSON, &v); err != nil {
		return v, err
	}
	if err := validateResources(v); err != nil {
		return v, err
	}
	return v, nil
}

func parseSingularityRequestJSON(rawJSON string) (SingularityRequestJSON, error) {
	v := SingularityRequestJSON{}
	err := jsonutil.StrictParseJSON(rawJSON, &v)
	return v, err
}

// SousResources returns the equivalent sous.Resources.
func (sr SingularityResources) SousResources() sous.Resources {
	r := make(sous.Resources, len(sr))
	for k, v := range sr {
		sousName, ok := resourceNameSingToSous[k]
		if !ok {
			sousName = k
		}
		r[sousName] = strconv.FormatFloat(v, 'g', -1, 64)
	}
	return r
}

var resourceNameSingToSous = map[string]string{
	"cpus":     "cpus",
	"numPorts": "ports",
	"memoryMb": "memory",
}

func validateResources(v SingularityJSON) error {
	seen := map[string]struct{}{}
	for k := range v.Resources {
		if _, ok := resourceNameSingToSous[k]; !ok {
			return fmt.Errorf("invalid resource name %q", k)
		}
		seen[k] = struct{}{}
	}
	var missing []string
	for k := range resourceNameSingToSous {
		if _, ok := seen[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) != 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing resource(s): %s", strings.Join(missing, ", "))
	}
	return nil
}
