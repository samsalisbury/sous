package config

import (
	"fmt"

	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

// DeployFilterFlags are CLI flags used to configure the underlying deployments
// a given command will refer to
// N.b. that not every command will use every filter
type DeployFilterFlags struct {
	Source   string
	Repo     string
	Offset   string
	Flavor   string
	Tag      string
	Revision string
	Cluster  string
	All      bool
}

// BuildPredicate returns a predicate used for filtering targeted deployments.
//
// It returns an error if the combination of flags is invalid, or if parseSL
// returns an error parsing Source.
func (f *DeployFilterFlags) BuildPredicate(parseSL func(string) (sous.SourceLocation, error)) (sous.DeploymentPredicate, error) {
	var preds []sous.DeploymentPredicate

	if f.Source != "" {
		if f.Repo != "" {
			return nil, fmt.Errorf("you cannot specify both -source and -repo")
		}
		if f.Offset != "" {
			return nil, fmt.Errorf("you cannot specify both -source and -offset")
		}
		if f.All {
			return nil, fmt.Errorf("you cannot specify both -source and -all")
		}
		sl, err := parseSL(f.Source)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing -source flag")
		}
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.Location == sl
		})
	}

	if f.All {
		if f.Repo != "" {
			return nil, fmt.Errorf("you cannot specify both -all and -repo")
		}
		if f.Offset != "" {
			return nil, fmt.Errorf("you cannot specify both -all and -offset")
		}
		if f.Flavor != "" {
			return nil, fmt.Errorf("you cannot specify both -all and -flavor")
		}
		return func(*sous.Deployment) bool { return true }, nil
	}

	if f.Repo != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.Location.Repo == f.Repo
		})
	}

	if f.Offset != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.Location.Dir == f.Offset
		})
	}

	if f.Flavor != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.Flavor == f.Flavor
		})
	}

	if f.Tag != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.Tag() == f.Tag
		})
	}

	if f.Revision != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.RevID() == f.Revision
		})
	}

	if f.Cluster != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.ClusterName == f.Cluster
		})
	}

	switch len(preds) {
	case 0:
		return nil, nil
	case 1:
		return preds[0], nil
	default:
		return func(d *sous.Deployment) bool {
			for _, f := range preds {
				if !f(d) { // AND(preds...)
					return false
				}
			}
			return true
		}, nil
	}
}
