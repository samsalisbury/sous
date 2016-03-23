package cli

import (
	"flag"

	"github.com/opentable/sous/util/cmdr"
)

type SousBuild struct {
	Sous         *Sous
	WDShell      LocalWorkDirShell
	ScratchShell ScratchDirShell
	flags        struct {
		target              string
		rebuild, rebuildAll bool
	}
}

const sousBuildHelp = `
build your project

build builds the project in your current directory by default. If you pass it a
path, it will instead build the project at that path.

args: [path]
`

func (*SousBuild) Help() *cmdr.Help { return cmdr.ParseHelp(sousBuildHelp) }

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
	//build := sous.NewBuild(sb.WDShell.Dir, sb.ScratchShell.Dir)

	return cmdr.InternalError(nil, "not implemented")
}
