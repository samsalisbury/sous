package cli

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/firsterr"
)

// SousBuild is the command description for `sous build`
// Implements cmdr.Command, cmdr.Executor and cmdr.AddFlags
type SousBuild struct {
	BuildContextFunc
	LabellerFunc
	RegistrarFunc
	Selector sous.Selector
	flags    struct {
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

// Help returns the help string for this command
func (*SousBuild) Help() string { return sousBuildHelp }

// Execute fulfills the cmdr.Executor interface
func (sb *SousBuild) Execute(args []string) cmdr.Result {
	var (
		bc        *sous.BuildContext
		labeller  sous.Labeller
		registrar sous.Registrar
	)
	err := firsterr.Set(
		func(err *error) { bc, *err = sb.BuildContextFunc() },
		func(err *error) { labeller, *err = sb.LabellerFunc() },
		func(err *error) { registrar, *err = sb.RegistrarFunc() },
	)
	if err != nil {
		return EnsureErrorResult(err)
	}
	if len(args) != 0 {
		path := args[0]
		if err := bc.Sh.CD(path); err != nil {
			return cmdr.EnsureErrorResult(err)
		}
	}

	mgr := &sous.BuildManager{
		BuildConfig: &sb.flags.config,
		Selector:    sb.Selector,
		Labeller:    labeller,
		Registrar:   registrar,
	}
	mgr.BuildConfig.Context = bc

	result, err := mgr.Build()

	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	//	return Success(result)
	return Success(result)
}
