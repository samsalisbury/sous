package cli

import (
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousBuild is the command description for `sous build`
// Implements cmdr.Command, cmdr.Executor and cmdr.AddFlags
type SousBuild struct {
	Sous          *Sous
	DockerClient  LocalDockerClient
	Config        LocalSousConfig
	WDShell       LocalWorkDirShell
	ScratchShell  ScratchDirShell
	SourceContext *sous.SourceContext
	BuildContext  *sous.BuildContext
	Builder       sous.Builder
	flags         struct {
		target              string
		rebuild, rebuildAll bool
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
	if len(args) != 0 {
		path := args[0]
		if err := sb.WDShell.CD(path); err != nil {
			return cmdr.EnsureErrorResult(err)
		}
	}

	bp := docker.NewDockerfileBuildpack()
	dr, err := bp.Detect(sb.BuildContext)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	result, err := sb.Builder.Build(sb.BuildContext, bp, dr)

	//nc := sous.NewNameCache(sb.DockerClient, sb.Config.DatabaseDriver, sb.Config.DatabaseConnection)

	//_, err := sous.RunBuild(nc, "docker.otenv.com",
	//	sb.SourceContext, sb.WDShell, sb.ScratchShell)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	//	return Success(result)
	return Success(result)
}
