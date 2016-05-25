package cli

import (
	"flag"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/cmdr"
)

type SousBuild struct {
	Sous          *Sous
	WDShell       LocalWorkDirShell
	ScratchShell  ScratchDirShell
	SourceContext *sous.SourceContext
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

func (*SousBuild) Help() string { return sousBuildHelp }

func (sb *SousBuild) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&sb.flags.target, "target", "app",
		"build a specific target")
	fs.BoolVar(&sb.flags.rebuild, "rebuild", false,
		"force a rebuild of the top-level target")
	fs.BoolVar(&sb.flags.rebuildAll, "rebuild-all", false,
		"similar to rebuild, but also rebuilds all transitive dependencies")
}

func (sb *SousBuild) Execute(args []string) cmdr.Result {
	if len(args) != 0 {
		path := args[0]
		if err := sb.WDShell.CD(path); err != nil {
			return cmdr.EnsureErrorResult(err)
		}
	}
	dbp := docker.NewDockerfileBuildpack("docker.otenv.com")
	build, err := sous.NewBuildWithShells(dbp, sb.SourceContext, sb.WDShell.Sh, sb.ScratchShell.Sh)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	result, err := build.Start()
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	return Success(result)
}
