package cli

import (
	"flag"

	"github.com/opentable/sous/ext/otpl"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousInit is the command description for `sous init`
type SousInit struct {
	DeployFilterFlags
	Flags         OTPLFlags
	SourceContext *sous.SourceContext
	WD            LocalWorkDirShell
	GDM           CurrentGDM
	State         *sous.State
	StateWriter   LocalStateWriter
}

func init() { TopLevelCommands["init"] = &SousInit{} }

const sousInitHelp = `
initialise a new sous project

usage: sous init

Sous init uses contextual information from your current source code tree and
repository to generate a basic configuration for that project. You will need to
flesh out some additional details.
`

// Help returns the help string for this command
func (si *SousInit) Help() string { return sousInitHelp }

// RegisterOn adds flag sets for sous init to the dependency injector.
func (si *SousInit) RegisterOn(psy Addable) {
	// Add a zero DepoyFilterFlags to the graph, as we assume a clean build.
	psy.Add(&si.DeployFilterFlags)
	psy.Add(&si.Flags)
}

// AddFlags adds the flags for sous init.
func (si *SousInit) AddFlags(fs *flag.FlagSet) {
	if err := AddFlags(fs, &si.Flags, otplFlagsHelp); err != nil {
		panic(err)
	}
}

// Execute fulfills the cmdr.Executor interface
func (si *SousInit) Execute(args []string) cmdr.Result {
	ctx := si.SourceContext
	sourceLocation := ctx.SourceLocation()

	ds := si.GDM.Filter(func(d *sous.Deployment) bool {
		return d.SourceID.Location() == sourceLocation
	})
	if ds.Len() != 0 {
		return UsageErrorf("init failed: manifest %q already exists", sourceLocation)
	}

	var deploySpecs, otplDeploySpecs sous.DeploySpecs
	if !si.Flags.IgnoreOTPLDeploy {
		otplParser := otpl.NewDeploySpecParser()
		otplDeploySpecs = otplParser.GetDeploySpecs(si.WD.Sh)
	}
	if !si.Flags.UseOTPLDeploy && !si.Flags.IgnoreOTPLDeploy && len(otplDeploySpecs) != 0 {
		return UsageErrorf("otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed")
	}
	if si.Flags.UseOTPLDeploy {
		if len(otplDeploySpecs) == 0 {
			return UsageErrorf("you specified -use-otpl-deploy, but no valid deployments were found in config/")
		}
		deploySpecs = sous.DeploySpecs{}
		for clusterName, spec := range otplDeploySpecs {
			if _, ok := si.State.Defs.Clusters[clusterName]; !ok {
				sous.Log.Warn.Printf("otpl-deploy config for cluster %q ignored", clusterName)
				continue
			}
			deploySpecs[clusterName] = spec
		}
	}
	if len(deploySpecs) == 0 {
		deploySpecs = defaultDeploySpecs()
	}

	m := &sous.Manifest{
		Source:      sourceLocation,
		Deployments: deploySpecs,
	}

	if ok := si.State.Manifests.Add(m); !ok {
		return UsageErrorf("manifest %q already exists", m.ID())
	}

	if err := si.StateWriter.WriteState(si.State); err != nil {
		return EnsureErrorResult(err)
	}

	return SuccessYAML(m)
}

func defaultDeploySpecs() sous.DeploySpecs {
	return sous.DeploySpecs{
		"Global": {
			DeployConfig: sous.DeployConfig{
				Resources:    sous.Resources{},
				Env:          map[string]string{},
				NumInstances: 3,
			},
		},
	}
}
