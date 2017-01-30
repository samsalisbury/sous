// Package otpl adds some OpenTable-specific interop methods. These will one day
// be removed.
package otpl

import (
	"path"
	"strconv"
	"sync"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
)

type (
	// ManifestParser parses sous.DeploySpecs from otpl-deploy config files.
	// NOTE: otpl-deploy config is an internal tool at OpenTable, one day this
	// code will be removed.
	ManifestParser struct {
		debugf func(string, ...interface{})
		debug  func(...interface{})
		WD     shell.Shell
	}
	// SingularityJSON represents the JSON in an otpl-deploy singularity.json
	// file.
	SingularityJSON struct {
		Resources SingularityResources
		Env       sous.Env
	}
	// SingularityResources represents the resources section in SingularityJSON.
	SingularityResources map[string]float64
	// SingularityRequestJSON represents JSON in an otpl-deploy
	// singularity-request.json file.
	SingularityRequestJSON struct {
		// Instances is the number of instances in this deployment.
		Instances int
		// Owners is a comma-separated list of email addresses.
		Owners []string
		// NOTE: We do not currently support Daemon, RackSensitive or LoadBalanced
		//Daemon, RackSensitive, LoadBalanced bool
	}
)

// SousResources returns the equivalent sous.Resources.
func (sr SingularityResources) SousResources() sous.Resources {
	r := make(sous.Resources, len(sr))
	for k, v := range sr {
		if k == "numPorts" {
			k = "ports"
		}
		if k == "memoryMb" {
			k = "memory"
		}
		r[k] = strconv.FormatFloat(v, 'g', -1, 64)
	}
	return r
}

// NewManifestParser generates a new ManifestParser with default logging.
func NewManifestParser() *ManifestParser {
	return &ManifestParser{debugf: sous.Log.Debug.Printf, debug: sous.Log.Debug.Println}
}

type otplDeployConfig struct {
	// Name is either "<cluster>" or "<cluster>.<flavor>".
	// It is unique for all OTPL configs in a single project.
	Name   string
	Owners []string
	Spec   *sous.DeploySpec
}

// ParseManifests searches the working directory of wd to find otpl-deploy
// config files in their standard locations (config/{cluster-name}), and
// converts them to sous.DeploySpecs.
func (mp *ManifestParser) ParseManifests(wd shell.Shell) *sous.Manifest {
	wd = wd.Clone()
	if err := wd.CD("config"); err != nil {
		return nil
	}
	l, err := wd.List()
	if err != nil {
		mp.debug(err)
		return nil
	}
	c := make(chan *otplDeployConfig)
	wg := sync.WaitGroup{}
	wg.Add(len(l))
	go func() { wg.Wait(); close(c) }()
	for _, f := range l {
		f := f
		go func() {
			defer wg.Done()
			if !f.IsDir() {
				return
			}
			wd := wd.Clone()
			if err := wd.CD(f.Name()); err != nil {
				mp.debug(err)
				return
			}
			if otplConfig := mp.parseSingleOTPLConfig(wd); otplConfig != nil {
				c <- otplConfig
			}
		}()
	}
	deployConfigs := sous.DeploySpecs{}
	owners := sous.NewOwnerSet()
	for s := range c {
		deployConfigs[s.Name] = *s.Spec
		for _, o := range s.Owners {
			owners.Add(o)
		}
	}
	return &sous.Manifest{
		Deployments: deployConfigs,
		Owners:      owners.Slice(),
	}
}

// ParseSingleOTPLConfig returns a single sous.DeploySpec from the working
// directory of wd. It assumes that this directory contains at least a file
// called singularity.json, and optionally an additional file called
// singularity-requst.json.
func (mp *ManifestParser) parseSingleOTPLConfig(wd shell.Shell) *otplDeployConfig {
	v := SingularityJSON{}
	if !wd.Exists("singularity.json") {
		mp.debugf("no singularity.json in %s", wd.Dir())
		return nil
	}
	if err := wd.JSON(&v, "cat", "singularity.json"); err != nil {
		mp.debugf("error reading %s: %s", path.Join(wd.Dir(),
			"singularity.json"), err)
		return nil
	}
	deploySpec := &otplDeployConfig{
		Name: path.Base(wd.Dir()),
		Spec: &sous.DeploySpec{
			DeployConfig: sous.DeployConfig{
				Resources: v.Resources.SousResources(),
				Env:       v.Env,
			},
		},
	}
	request := SingularityRequestJSON{}
	if !wd.Exists("singularity-request.json") {
		mp.debugf("no singularity-request.json in %s", wd.Dir())
		return deploySpec
	}
	mp.debugf("%s/singularity-request.json exists, parsing it", wd.Dir())
	if err := wd.JSON(&request, "cat", "singularity-request.json"); err != nil {
		mp.debugf("error reading singularity-request.json: %s", err)
		return deploySpec
	}
	deploySpec.Spec.NumInstances = request.Instances
	deploySpec.Owners = request.Owners
	return deploySpec
}
