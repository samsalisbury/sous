package sous

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
)

//go:generate ggen cmap.CMap(cmap.go) sous.Manifests(manifests.go) CMKey:ManifestID Value:*Manifest

type (
	// Manifest is a minimal representation of the global deployment state of
	// a particular named application. It is designed to be written and read by
	// humans as-is, and expanded into full Deployments internally. It is a DTO,
	// which can be stored in YAML files.
	//
	// Manifest has a direct two-way mapping to/from Deployments.
	Manifest struct {
		// Source is the location of the source code for this piece of software.
		Source SourceLocation `validate:"nonzero"`
		// Flavor is an optional string, used to allow a single SourceLocation
		// to have multiple deployments defined per cluster. The empty Flavor
		// is perfectly valid. The pair (SourceLocation, Flavor) identifies a
		// manifest.
		Flavor string `yaml:",omitempty"`
		// Owners is a list of named owners of this repository. The type of this
		// field is subject to change.
		Owners []string
		// Kind is the kind of software that SourceRepo represents.
		Kind ManifestKind `validate:"nonzero"`
		// Deployments is a map of cluster names to DeploymentSpecs
		Deployments DeploySpecs `validate:"keys=nonempty,values=nonzero"`
	}
)

// ID returns the SourceLocation.
func (m Manifest) ID() ManifestID {
	return ManifestID{Source: m.Source, Flavor: m.Flavor}
}

// SetID sets the Source and Flavor fields of this Manifest to those of the
// supplied ManifestID.
func (m *Manifest) SetID(mid ManifestID) {
	m.Source = mid.Source
	m.Flavor = mid.Flavor
}

// Clone returns a deep copy of this Manifest.
func (m Manifest) Clone() (c *Manifest) {
	owners := make([]string, len(m.Owners))
	copy(m.Owners, owners)
	deployments := make(DeploySpecs, len(m.Deployments))
	for k, v := range m.Deployments {
		deployments[k] = v.Clone()
	}
	m.Owners = owners
	m.Deployments = deployments
	return &m
}

// FileLocation returns the path that the manifest should be saved to.
func (m *Manifest) FileLocation() string {
	return filepath.Join(string(m.Source.Repo), string(m.Source.Dir))
}

// Diff returns true and a list of differences if m and o are not equal.
// Otherwise returns false and nil.
func (m *Manifest) Diff(o *Manifest) (bool, []string) {
	if m == o {
		// They are the same pointer.
		return false, nil
	}
	var diffs []string
	diff := func(format string, a ...interface{}) { diffs = append(diffs, fmt.Sprintf(format, a...)) }
	if m.Source != o.Source {
		diff("source; this: %q; other: %q", m.Source, o.Source)
	}
	if m.Kind != o.Kind {
		diff("kind; this: %q; other: %q", m.Kind, o.Kind)
	}
	if len(m.Owners) != len(o.Owners) {
		diff("number of owners; this: %d; other: %d", len(m.Owners), len(o.Owners))
	} else {
		for i, owner := range m.Owners {
			if o.Owners[i] != owner {
				diff("owner in position %d; this: %d; other: %d", i, owner, o.Owners[i])
			}
		}
	}
	if len(m.Deployments) != len(o.Deployments) {
		diff("number of deployments; this: %d; other: %d", len(m.Deployments), len(o.Deployments))
	} else {
		for clusterName, deploySpec := range m.Deployments {
			_, differences := deploySpec.Diff(o.Deployments[clusterName])
			for _, deploySpecDiff := range differences {
				diff("%s: "+deploySpecDiff, clusterName)
			}
		}
	}
	return len(diffs) != 0, diffs
}

// Equal returns true iff o is equal to m.
func (m *Manifest) Equal(o *Manifest) bool {
	diff, _ := m.Diff(o)
	return !diff
}

// Validate implements Flawed for State
func (m *Manifest) Validate() []Flaw {
	var flaws []Flaw
	if m.Kind == "" {
		flaws = append(flaws, NewFlaw(
			fmt.Sprintf("manifest %q missing Kind", m.ID()),
			func() error { m.Kind = ManifestKindService; return nil },
		))
	} else {
		flaws = append(flaws, m.Kind.Validate()...)
	}

	for cluster, d := range m.Deployments {
		df := d.Validate()
		for _, f := range df {
			f.AddContext("cluster", cluster)
		}
		flaws = append(flaws, df...)
	}

	for _, f := range flaws {
		f.AddContext("manifest", m)
	}

	return flaws
}

// Repair implements Flawed for State
func (m *Manifest) Repair(fs []Flaw) error {
	return errors.Errorf("Can't do nuffin with flaws yet")
}
