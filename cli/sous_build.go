package cli

import (
	"flag"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousBuild is the command description for `sous build`
// Implements cmdr.Command, cmdr.Executor and cmdr.AddFlags
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

// Help returns the help string for this command
func (*SousBuild) Help() string { return sousBuildHelp }

// AddFlags fulfills the cmdr.AddFlags interface
func (sb *SousBuild) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&sb.flags.target, "target", "app",
		"build a specific target")
	fs.BoolVar(&sb.flags.rebuild, "rebuild", false,
		"force a rebuild of the top-level target")
	fs.BoolVar(&sb.flags.rebuildAll, "rebuild-all", false,
		"similar to rebuild, but also rebuilds all transitive dependencies")
}

// Execute fulfills the cmdr.Executor interface
func (sb *SousBuild) Execute(args []string) cmdr.Result {
	if len(args) != 0 {
		path := args[0]
		if err := sb.WDShell.CD(path); err != nil {
			return cmdr.EnsureErrorResult(err)
		}
	}

	result, err := sous.RunBuild("docker.otenv.com", sb.SourceContext, sb.WDShell, sb.ScratchShell)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	return Success(result)
}
