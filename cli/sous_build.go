package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

type (
	// SousBuild is the command description for `sous build`
	// Implements cmdr.Command, cmdr.Executor and cmdr.AddFlags
	SousBuild struct {
		DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
		PolicyFlags       config.PolicyFlags       `inject:"optional"`

		SousGraph *graph.SousGraph

		opts graph.BuildActionOpts
	}
)

func init() { TopLevelCommands["build"] = &SousBuild{} }

const sousBuildHelp = `build your project

build builds the project in your current directory by default. If you pass it a
path, it will instead build the project at that path.

args: [path]
`

// AddFlags adds flags to the build command.
func (sb *SousBuild) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sb.DeployFilterFlags, SourceFlagsHelp)
	fs.BoolVar(&sb.PolicyFlags.Strict, "strict", false, "require that the build be pristine")
	fs.BoolVar(&sb.PolicyFlags.Dev, "dev", false, "run build with developer options")
	//fs.BoolVar(&sb.PolicyFlags.ForceClone, "force-clone", false, "force a shallow clone of the codebase before build")
	// above is commented prior to impl.
}

// Help returns the help string for this command
func (*SousBuild) Help() string { return sousBuildHelp }

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar
func (sb *SousBuild) RegisterOn(psy Addable) {
	psy.Add(&sb.DeployFilterFlags)
	psy.Add(&sb.PolicyFlags)
}

// Execute fulfills the cmdr.Executor interface
func (sb *SousBuild) Execute(args []string) cmdr.Result {
	sb.opts.CLIArgs = args
	sb.opts.DFF = sb.DeployFilterFlags
	build, err := sb.SousGraph.GetBuild(sb.opts)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	if err := build.Do(); err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	return cmdr.Success(build.Result())
}
