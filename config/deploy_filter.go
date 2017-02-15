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

func (f *DeployFilterFlags) BuildFilter(parseSL func(string) (sous.SourceLocation, error)) (*sous.ResolveFilter, error) {
	rf := &sous.ResolveFilter{}

	rf.Repo = f.Repo
	rf.Cluster = f.Cluster
	rf.Tag = f.Tag
	rf.Revision = f.Revision

	rf.Offset = f.Offset
	rf.Flavor = f.Flavor

	if f.Source != "" {
		if f.Repo != "" {
			return nil, fmt.Errorf("you cannot specify both -source and -repo")
		}
		if f.Offset != "" {
			return nil, fmt.Errorf("you cannot specify both -source and -offset")
		}
		sl, err := parseSL(f.Source)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing -source flag")
		}
		rf.Repo = sl.Repo
		rf.Offset = sl.Dir
	}

	if f.All && !rf.All() {
		return nil, errors.Errorf("You cannot specify both -all and filtering options: %s", rf)
	}

	return rf, nil
}

// BuildPredicate returns a predicate used for filtering targeted deployments.
//
// It returns an error if the combination of flags is invalid, or if parseSL
// returns an error parsing Source.
func (f *DeployFilterFlags) BuildPredicate(parseSL func(string) (sous.SourceLocation, error)) (sous.DeploymentPredicate, error) {
	rf, err := f.BuildFilter(parseSL)
	if err != nil {
		return nil, err
	}

	return rf.FilterDeployment, nil
}
