package cli

import (
	"flag"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousBuild is the command description for `sous build`
// Implements cmdr.Command, cmdr.Executor and cmdr.AddFlags
type SousBuild struct {
	*sous.BuildContext
	sous.Labeller
	sous.Registrar
	Selector         sous.Selector
	DeploymentConfig DeployFilterFlags
	flags            struct {
		config sous.BuildConfig
	}
}

func init() { TopLevelCommands["build"] = &SousBuild{} }

const sousBuildHelp = `
build your project

build builds the project in your current directory by default. If you pass it a
path, it will instead build the project at that path.

args: [path]
`

func (sb *SousBuild) AddFlags(fs *flag.FlagSet) {
	err := AddFlags(fs, &sb.DeploymentConfig, sourceFlagsHelp)
	if err != nil {
		panic(err)
	}
}

// Help returns the help string for this command
func (*SousBuild) Help() string { return sousBuildHelp }

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar
func (sb *SousBuild) RegisterOn(psy Addable) {
	psy.Add(sb.DeploymentConfig)
}

// Execute fulfills the cmdr.Executor interface
func (sb *SousBuild) Execute(args []string) cmdr.Result {
	var bc *sous.BuildContext
	if len(args) != 0 {
		path := args[0]
		if err := bc.Sh.CD(path); err != nil {
			return cmdr.EnsureErrorResult(err)
		}
	}

	mgr := &sous.BuildManager{
		BuildConfig: &sb.flags.config,
		Selector:    sb.Selector,
		Labeller:    sb.Labeller,
		Registrar:   sb.Registrar,
	}
	mgr.BuildConfig.Context = bc

	result, err := mgr.Build()

	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	//	return Success(result)
	return Success(result)
}
