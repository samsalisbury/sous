package sous

import (
	"bytes"
	"fmt"

	"github.com/opentable/sous/util/logging"
)

// ManifestID identifies a manifest by its SourceLocation and optional Flavor.
type ManifestID struct {
	// Source is the SourceLocation of deployments described in this Manifest.
	Source SourceLocation
	// Flavor is an optional string which can be appended if multiple different
	// deployments of code from this SourceLocation need to be deployed in the
	// same cluster.
	Flavor string
}

// FlavorSeparator separates the flavor part of a ManifestID from the
// SourceLocation part.
const FlavorSeparator = "~"

// ParseManifestID parses a ManifestID from a SourceLocation.
func ParseManifestID(s string) (ManifestID, error) {
	var mid ManifestID
	err := mid.UnmarshalText([]byte(s))
	return mid, err
}

// MustParseManifestID wraps ParseManifestID and panics if it returns an error.
func MustParseManifestID(s string) ManifestID {
	mid, err := ParseManifestID(s)
	if err != nil {
		panic(err)
	}
	return mid
}

func (mid ManifestID) String() string {
	f := ""
	if mid.Flavor != "" {
		f = FlavorSeparator + mid.Flavor
	}
	return mid.Source.String() + f
}

// QueryMap returns a map suitable for use with the HTTP API.
func (mid ManifestID) QueryMap() map[string]string {
	manifestQuery := map[string]string{}
	manifestQuery["repo"] = mid.Source.Repo
	manifestQuery["offset"] = mid.Source.Dir
	manifestQuery["flavor"] = mid.Flavor
	return manifestQuery
}

// MarshalText implements encoding.TextMarshaler.
// This is important for serialising maps that use ManifestID as a key.
func (mid ManifestID) MarshalText() ([]byte, error) {
	return []byte(mid.String()), nil
}

// UnmarshalText implements encoding.TextMarshaler.
// This is important for deserialising maps that use ManifestID as a key.
func (mid *ManifestID) UnmarshalText(b []byte) error {
	parts := bytes.Split(b, []byte(FlavorSeparator))
	if len(parts) > 2 {
		return fmt.Errorf("illegal manifest ID %q (contains more than one colon)", string(b))
	}
	if err := mid.Source.UnmarshalText(parts[0]); err != nil {
		return err
	}
	if len(parts) == 2 {
		mid.Flavor = string(parts[1])
	}
	return nil
}

// MarshalYAML serializes this ManifestID to a YAML string.
// It implements the github.com/go-yaml/yaml.Marshaler interface.
func (mid ManifestID) MarshalYAML() (interface{}, error) {
	return mid.String(), nil
}

// UnmarshalYAML deserializes a YAML string into this ManifestID.
// It implements the github.com/go-yaml/yaml.Unmarshaler interface.
func (mid *ManifestID) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	return mid.UnmarshalText([]byte(s))
}

// EachField implements logging.EachFielder on DeploymentID.
func (mid ManifestID) EachField(fn logging.FieldReportFn) {
	fn(logging.SousManifestId, mid.String())
	mid.Source.EachField(fn)
}
