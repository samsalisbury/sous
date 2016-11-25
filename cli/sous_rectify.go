package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousRectify is the injectable command object used for `sous rectify`.
type SousRectify struct {
	Config      graph.LocalSousConfig
	State       *sous.State
	GDM         graph.CurrentGDM
	SourceFlags config.DeployFilterFlags
	Resolver    *sous.Resolver
	flags       struct {
		dryrun                string
		repo, offset, cluster string
		all                   bool
	}
}

func init() { TopLevelCommands["rectify"] = &SousRectify{} }

const sousRectifyHelp = `
force Sous to make the deployment match the contents of the local state directory

usage: sous rectify

Several predicates are available to constrain the action of the rectification.
-repo, -offset and -cluster limit the rectification appropriately. When used
together, the result is the intersection of their images - that is, the
conditions are "anded." By implication, each can only be used once.
NOTE: the successful use of these predicates requires all-team coordination.
Use with great care.

Because of the hazard involved in doing complete rectification at the command
line, sous rectify requires the -all flag to consider the whole tree. This is
almost certainly not what you want. Even if it is, you certainly want to trial
your rectifies with -dry-run=scheduler first.

Note: by default this command will query a live docker registry and make
changes to live Singularity clusters.
`

// Help returns the help string.
func (*SousRectify) Help() string { return sousRectifyHelp }

// AddFlags adds flags for sous rectify.
func (sr *SousRectify) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sr.SourceFlags, RectifyFilterFlagsHelp)

	fs.StringVar(&sr.flags.dryrun, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
}

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar.
func (sr *SousRectify) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunOption(sr.flags.dryrun))
	psy.Add(&sr.SourceFlags)
}

// Execute fulfils the cmdr.Executor interface.
func (sr *SousRectify) Execute(args []string) cmdr.Result {
	if !sr.SourceFlags.All && sr.Resolver.ResolveFilter.All() {
		return UsageErrorf("Please specify what to rectify using the -repo tag.\n" +
			"(Or -all if you really mean to rectify the whole world; see 'sous help rectify'.)")
	}

	if err := sr.Resolver.Resolve(sr.GDM.Clone(), sr.State.Defs.Clusters); err != nil {
		return EnsureErrorResult(err)
	}

	return Success()
}
