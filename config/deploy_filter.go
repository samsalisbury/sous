package config

import (
	"fmt"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

// DeployFilterFlags are CLI flags used to configure the underlying deployments
// a given command will refer to
// N.b. that not every command will use every filter
type DeployFilterFlags struct {
	DeploymentIDFlags
	SourceVersionFlags
	All bool
}

// MakeDeployFilterFlags encapsulates setting fields on a brand new
// DeployFilterFlags. It is more convenient to use this than to manually
// construct the hierarchical nested structs otherwise necessary.
func MakeDeployFilterFlags(config func(*DeployFilterFlags)) DeployFilterFlags {
	dff := DeployFilterFlags{}
	config(&dff)
	return dff
}

// SourceVersionFlags are Tag and Revision.
type SourceVersionFlags struct {
	Tag      string
	Revision string
}

// SourceIDFlags identify a version of a particular SourceLocation.
type SourceIDFlags struct {
	SourceLocationFlags
	SourceVersionFlags
}

// SourceID returns the sous.SourceID represented by these flags.
func (f SourceIDFlags) SourceID() (sous.SourceID, error) {
	version, err := semv.Parse(f.Tag)
	if err != nil {
		return sous.SourceID{}, err
	}
	return sous.SourceID{
		Version:  version,
		Location: f.SourceLocationFlags.SourceLocation(),
	}, nil
}

// DeploymentIDFlags identify a Deployment.
type DeploymentIDFlags struct {
	ManifestIDFlags
	Cluster string
}

// ManifestIDFlags identify a manifest.
type ManifestIDFlags struct {
	SourceLocationFlags
	Flavor string
}

// SourceLocationFlags identify a SourceLocation.
type SourceLocationFlags struct {
	Source string
	Repo   string
	Offset string
}

// SourceLocation returns the SourceLocation represented by these flags.
func (f SourceLocationFlags) SourceLocation() sous.SourceLocation {
	return sous.SourceLocation{
		Repo: f.Repo,
		Dir:  f.Offset,
	}
}

// BuildFilter creates a ResolveFilter from DeployFilterFlags.
func (f *DeployFilterFlags) BuildFilter(parseSL func(string) (sous.SourceLocation, error)) (*sous.ResolveFilter, error) {
	rf := &sous.ResolveFilter{}

	rf.Repo = buildFieldMatcher(f.Repo, true)
	rf.Cluster = buildFieldMatcher(f.Cluster, true)
	rf.Revision = buildFieldMatcher(f.Revision, true)
	rf.Offset = buildFieldMatcher(f.Offset, f.All)
	rf.Flavor = buildFieldMatcher(f.Flavor, f.All)

	if !f.All && f.Tag != "" && f.Tag != "*" {
		err := rf.SetTag(f.Tag)
		if err != nil {
			return nil, err
		}
	}

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
		rf.Repo = sous.NewResolveFieldMatcher(sl.Repo)
		rf.Offset = sous.NewResolveFieldMatcher(sl.Dir)
	}

	if f.All && !rf.All() {
		return nil, errors.Errorf("You cannot specify both -all and filtering options: %s", rf)
	}

	return rf, nil
}

// EachField implements logging.EachFielder on DeployFilterFlags.
func (f DeployFilterFlags) EachField(fn func(logging.FieldName, interface{})) {
	fn(logging.FilterCluster, f.Cluster)
	fn(logging.FilterFlavor, f.Flavor)
	fn(logging.FilterOffset, f.Offset)
	fn(logging.FilterRepo, f.Repo)
	fn(logging.FilterRevision, f.Revision)
	fn(logging.FilterTag, f.Tag)
}

func buildFieldMatcher(config string, all bool) sous.ResolveFieldMatcher {
	if config == "*" || (all && config == "") {
		return sous.ResolveFieldMatcher{}
	}
	return sous.NewResolveFieldMatcher(config)
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
