// Package otpl adds some OpenTable-specific interop methods. These will one day
// be removed.
package otpl

import (
	"fmt"
	"path"

	"strings"

	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/shell"
)

// ManifestParser parses sous.DeploySpecs from otpl-deploy config files.
// NOTE: otpl-deploy config is an internal tool at OpenTable, one day this
// code will be removed.
type ManifestParser struct {
	Log logging.LogSink
	WD  shell.Shell
}

// NewManifestParser generates a new ManifestParser with default logging.
func NewManifestParser(ls logging.LogSink) *ManifestParser {
	return &ManifestParser{Log: ls}
}

type otplDeployConfig struct {
	// Name is "<cluster>".
	// It is unique for all OTPL configs in a single project by flavor.
	Name                   string
	RequestID, RequestType string
	Owners                 []string
	Spec                   *sous.DeploySpec
}

type otplDeployManifest struct {
	Kind   sous.ManifestKind
	Owners sous.OwnerSet
	Specs  sous.DeploySpecs
}

type otplDeployManifests map[string]*otplDeployManifest

func getDeployManifest(manifests otplDeployManifests, key string) *otplDeployManifest {
	if manifest, ok := manifests[key]; ok {
		return manifest
	}
	manifest := &otplDeployManifest{
		Owners: sous.OwnerSet{},
		Specs:  sous.DeploySpecs{},
	}
	manifests[key] = manifest
	return manifest
}

// ParseManifests searches the working directory of wd to find otpl-deploy
// config files in their standard locations (config/{cluster-name}] or
// config/{cluster-name}.{flavor}), and converts them to sous.DeploySpecs.
func (mp *ManifestParser) ParseManifests(wd shell.Shell) (sous.Manifests, error) {
	wd = wd.Clone()
	if err := wd.CD("config"); err != nil {
		return sous.NewManifests(), fmt.Errorf("entering config directory: %s", err)
	}
	l, err := wd.List()
	if err != nil {
		return sous.NewManifests(), fmt.Errorf("reading current directory: %s", err)
	}
	var c []*otplDeployConfig
	for _, f := range l {
		f := f
		if !f.IsDir() {
			continue
		}
		wd := wd.Clone()
		if err := wd.CD(f.Name()); err != nil {
			return sous.NewManifests(), fmt.Errorf("entering directory %q: %s", f.Name(), err)
		}
		otplConfig, err := mp.parseSingleOTPLConfigErr(wd)
		if err != nil {
			return sous.NewManifests(), fmt.Errorf("parsing otpl deploy config: %s", err)
		}
		c = append(c, otplConfig)
	}
	deployManifests := otplDeployManifests{}
	for _, s := range c {
		cluster, flavor := getClusterAndFlavor(s)
		deployManifest := getDeployManifest(deployManifests, flavor)
		deployManifest.Specs[cluster] = *s.Spec
		kind, ok := singularity.MapRequestTypeToManifestKind(s.RequestType)
		if !ok {
			return sous.NewManifests(), fmt.Errorf("invalid request type %q for deployment %q", s.RequestType, s.Name)
		}
		deployManifest.Kind = kind
		for _, o := range s.Owners {
			deployManifest.Owners.Add(o)
		}
	}
	manifests := sous.NewManifests()
	for flavor, dm := range deployManifests {
		manifests.Add(&sous.Manifest{
			Kind:        dm.Kind,
			Flavor:      flavor,
			Deployments: dm.Specs,
			Owners:      dm.Owners.Slice(),
		})
	}
	return manifests, nil
}

// GetClusterAndFlavor returns the cluster and flavor by extracting values
// from the otplDeployConfig name. The pattern is {cluster}.{flavor} as
// defined in the otpl scripts.
func getClusterAndFlavor(s *otplDeployConfig) (string, string) {
	splitName := strings.Split(s.Name, ".")
	cluster := splitName[0]
	flavor := ""
	if len(splitName) > 1 {
		flavor = splitName[1]
	}
	return cluster, flavor
}

func (mp *ManifestParser) parseSingleOTPLConfigErr(wd shell.Shell) (*otplDeployConfig, error) {
	if !wd.Exists("singularity.json") {
		return nil, fmt.Errorf("no singularity.json present")
	}
	rawJSON, err := wd.Stdout("cat", "singularity.json")
	if err != nil {
		return nil, fmt.Errorf("error reading singularity.json: %s", err)
	}

	v, err := parseSingularityJSON(rawJSON)
	if err != nil {
		return nil, fmt.Errorf("error parsing singularity.json: %s", err)
	}

	if v.Env == nil {
		v.Env = map[string](string){}
	}

	deploySpec := &otplDeployConfig{
		Name: path.Base(wd.Dir()),
		Spec: &sous.DeploySpec{
			DeployConfig: sous.DeployConfig{
				SingularityRequestID: v.RequestID,
				Resources:            v.Resources.SousResources(),
				Env:                  v.Env,
			},
		},
	}
	if !wd.Exists("singularity-request.json") {
		return nil, fmt.Errorf("no singularity-request.json present")
	}
	rawSRJSON, err := wd.Stdout("cat", "singularity-request.json")
	if err != nil {
		return nil, fmt.Errorf("reading singularity-request.json: %s", err)
	}

	request, err := parseSingularityRequestJSON(rawSRJSON)
	if err != nil {
		return nil, fmt.Errorf("error parsing singularity-request.json: %s", err)
	}

	deploySpec.Spec.NumInstances = request.Instances
	deploySpec.Owners = request.Owners
	deploySpec.RequestType = request.RequestType
	return deploySpec, nil
}

// ParseSingleOTPLConfig returns a single sous.DeploySpec from the working
// directory of wd. It assumes that this directory contains at least a file
// called singularity.json, and optionally an additional file called
// singularity-requst.json.
func (mp *ManifestParser) parseSingleOTPLConfig(wd shell.Shell) *otplDeployConfig {
	o, err := mp.parseSingleOTPLConfigErr(wd)
	if err != nil {
		messages.ReportLogFieldsMessage(err.Error(), logging.WarningLevel, mp.Log, wd.Dir())
		return nil
	}
	return o
}
