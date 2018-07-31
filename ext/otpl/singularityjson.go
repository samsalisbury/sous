package otpl

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	sous "github.com/opentable/sous/lib"
)

type (
	// SingularityJSON represents the JSON in an otpl-deploy singularity.json
	// file. Note that the json tags are essential to validating parsed JSON
	// contains only recognised fields.
	SingularityJSON struct {
		RequestID string               `json:"requestId"`
		Resources SingularityResources `json:"resources,omitempty"`
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
		Instances int `json:"instances,omitempty"`
		// Owners is a comma-separated list of email addresses.
		Owners []string `json:"owners,omitempty"`
	}
)

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

func strictParseJSON(rawJSON string, v interface{}) error {
	comp := map[string]interface{}{}
	if err := json.Unmarshal([]byte(rawJSON), v); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(rawJSON), &comp); err != nil {
		return err
	}
	compJSONb, err := json.Marshal(comp)
	if err != nil {
		return err
	}
	understoodJSONb, err := json.Marshal(v)
	if err != nil {
		return err
	}
	understoodJSON := string(understoodJSONb)
	compJSON := string(compJSONb)

	equal, err := equalJSON(compJSON, understoodJSON)
	if err != nil {
		return err
	}
	if !equal {
		return fmt.Errorf("unrecognised fields:\n%sunderstood:\n%s",
			compJSON, understoodJSON)
	}
	return nil
}

func equalJSON(a, b string) (bool, error) {
	var aVal, bVal interface{}
	if err := json.Unmarshal([]byte(a), &aVal); err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(b), &bVal); err != nil {
		return false, err
	}
	return reflect.DeepEqual(aVal, bVal), nil
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

func parseSingularityJSON(rawJSON string) (SingularityJSON, error) {
	v := SingularityJSON{}
	if err := strictParseJSON(rawJSON, &v); err != nil {
		return v, err
	}
	if err := validateResources(v); err != nil {
		return v, err
	}
	return v, nil
}

func parseSingularityRequestJSON(rawJSON string) (SingularityRequestJSON, error) {
	v := SingularityRequestJSON{}
	err := strictParseJSON(rawJSON, &v)
	return v, err
}
