// Package otpl adds some OpenTable-specific interop methods. These will one day
// be removed.
package otpl

import (
	"log"
	"path"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
)

type (
	DeploySpecParser struct {
		debugf func(string, ...interface{})
		debug  func(...interface{})
		WD     shell.Shell
	}
	SingularityJSON struct {
		Resources sous.Resources
		Env       sous.Env
	}

	SingularityRequestJSON struct {
		Instances int
		// NOTE: Owners are not supported at DeploySpec level, only at Manifest
		// level... Maybe that needs to change.
		//Owners                              []string
		// NOTE: We do not currently support Daemon, RackSensitive or LoadBalanced
		//Daemon, RackSensitive, LoadBalanced bool
	}
)

func NewDeploySpecParser() *DeploySpecParser {
	return &DeploySpecParser{debugf: log.Printf, debug: log.Println}
}

func (dsp *DeploySpecParser) GetDeploySpecs(wd shell.Shell) sous.DeploySpecs {
	wd = wd.Clone()
	if err := wd.CD("config"); err != nil {
		return nil
	}
	l, err := wd.List()
	if err != nil {
		return nil
	}
	deployConfigs := sous.DeploySpecs{}
	for _, f := range l {
		if !f.IsDir() {
			continue
		}
		wd := wd.Clone()
		if err := wd.CD(f.Name()); err != nil {
			dsp.debug(err)
			continue
		}
		if otplConfig := dsp.GetSingleDeploySpec(wd); otplConfig != nil {
			name := path.Base(wd.Dir())
			deployConfigs[name] = *otplConfig
		}
	}
	return deployConfigs
}

func (dsp *DeploySpecParser) GetSingleDeploySpec(wd shell.Shell) *sous.PartialDeploySpec {
	v := SingularityJSON{}
	if err := wd.JSON(&v, "cat", "singularity.json"); err != nil {
		return nil
	}
	deploySpec := sous.PartialDeploySpec{
		DeployConfig: sous.DeployConfig{
			Resources: v.Resources,
			Env:       v.Env,
		},
	}
	request := SingularityRequestJSON{}
	if !wd.Exists("singularity-request.json") {
		dsp.debugf("%s/singularity-request.json not found", wd.Dir())
		return &deploySpec
	}
	dsp.debugf("%s/singularity-request.json exists, parsing it", wd.Dir())
	if err := wd.JSON(&request, "cat", "singularity-request.json"); err != nil {
		dsp.debugf("error parsing singularity-request.json: %s", err)
		return &deploySpec
	}
	deploySpec.NumInstances = request.Instances
	return &deploySpec
}
