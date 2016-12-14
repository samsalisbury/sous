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
	// DeploySpecParser parses sous.DeploySpecs from otpl-deploy config files.
	// NOTE: otpl-deploy config is an internal tool at OpenTable, one day this
	// code will be removed.
	DeploySpecParser struct {
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
		// NOTE: Owners are not supported at DeploySpec level, only at Manifest
		// level... Maybe that needs to change.
		//Owners                              []string
		// NOTE: We do not currently support Daemon, RackSensitive or LoadBalanced
		//Daemon, RackSensitive, LoadBalanced bool
	}
)

// SousResources returns the equivalent sous.Resources.
func (sr SingularityResources) SousResources() sous.Resources {
	r := make(sous.Resources, len(sr))
	for k, v := range sr {
		r[k] = strconv.FormatFloat(v, 'g', -1, 64)
	}
	return r
}

// NewDeploySpecParser generates a new DeploySpecParser with default logging.
func NewDeploySpecParser() *DeploySpecParser {
	return &DeploySpecParser{debugf: sous.Log.Debug.Printf, debug: sous.Log.Debug.Println}
}

type namedDeploySpec struct {
	Name string
	Spec *sous.DeploySpec
}

// GetDeploySpecs searches the working directory of wd to find otpl-deploy
// config files in their standard locations (config/{cluster-name}), and
// converts them to sous.DeploySpecs.
func (dsp *DeploySpecParser) GetDeploySpecs(wd shell.Shell) sous.DeploySpecs {
	wd = wd.Clone()
	if err := wd.CD("config"); err != nil {
		return nil
	}
	l, err := wd.List()
	if err != nil {
		dsp.debug(err)
		return nil
	}
	c := make(chan namedDeploySpec)
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
				dsp.debug(err)
				return
			}
			if otplConfig := dsp.GetSingleDeploySpec(wd); otplConfig != nil {
				name := path.Base(wd.Dir())
				c <- namedDeploySpec{name, otplConfig}
			}
		}()
	}
	deployConfigs := sous.DeploySpecs{}
	for s := range c {
		deployConfigs[s.Name] = *s.Spec
	}
	return deployConfigs
}

// GetSingleDeploySpec returns a single sous.DeploySpec from the working
// directory of wd. It assumes that this directory contains at least a file
// called singularity.json, and optionally an additional file called
// singularity-requst.json.
func (dsp *DeploySpecParser) GetSingleDeploySpec(wd shell.Shell) *sous.DeploySpec {
	v := SingularityJSON{}
	if !wd.Exists("singularity.json") {
		dsp.debugf("no singularity.json in %s", wd.Dir())
		return nil
	}
	if err := wd.JSON(&v, "cat", "singularity.json"); err != nil {
		dsp.debugf("error reading %s: %s", path.Join(wd.Dir(),
			"singularity.json"), err)
		return nil
	}
	deploySpec := sous.DeploySpec{
		DeployConfig: sous.DeployConfig{
			Resources: v.Resources.SousResources(),
			Env:       v.Env,
		},
	}
	request := SingularityRequestJSON{}
	if !wd.Exists("singularity-request.json") {
		dsp.debugf("no singularity-request.json in %s", wd.Dir())
		return &deploySpec
	}
	dsp.debugf("%s/singularity-request.json exists, parsing it", wd.Dir())
	if err := wd.JSON(&request, "cat", "singularity-request.json"); err != nil {
		dsp.debugf("error reading singularity-request.json: %s", err)
		return &deploySpec
	}
	deploySpec.NumInstances = request.Instances
	return &deploySpec
}
