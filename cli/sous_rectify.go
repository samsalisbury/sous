package cli

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousRectify is the injectable command object used for `sous rectify`
type SousRectify struct {
	Config       LocalSousConfig
	DockerClient LocalDockerClient
	Deployer     sous.Deployer
	Registry     sous.Registry
	State        *sous.State
	GDM          CurrentGDM
	SourceFlags  DeployFilterFlags
	flags        struct {
		dryrun,
		repo, offset, cluster string
		all bool
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

// Help returns the help string
func (*SousRectify) Help() string { return sousRectifyHelp }

// AddFlags adds flags for sous rectify
func (sr *SousRectify) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sr.SourceFlags, rectifyFilterFlagsHelp)

	fs.StringVar(&sr.flags.dryrun, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
}

// Execute fulfils the cmdr.Executor interface
func (sr *SousRectify) Execute(args []string) cmdr.Result {

	r, err := sr.buildResolver()
	if err != nil {
		return EnsureErrorResult(err)
	}
	if err := r.Resolve(sr.GDM.Clone(), sr.State.Defs.Clusters); err != nil {
		return EnsureErrorResult(err)
	}

	return Success()
}

func (sr *SousRectify) buildResolver() (*sous.Resolver, error) {
	sr.resolveDryRunFlag(sr.flags.dryrun)

	predicate := sr.SourceFlags.buildPredicate()

	if predicate == nil {
		return nil, fmt.Errorf("Cowardly refusing rectify with neither contraint nor `-all`! (see `sous help rectify`)")
	}

	return sous.NewResolver(sr.Deployer, sr.Registry, predicate), nil
}

func (sr *SousRectify) resolveDryRunFlag(dryrun string) {
	if dryrun == "both" || dryrun == "registry" {
		sr.Registry = sous.NewDummyRegistry()
	}
	if dryrun == "both" || dryrun == "scheduler" {
		drc := singularity.NewDummyRectificationClient(sr.Registry)
		drc.SetLogger(log.New(os.Stdout, "rectify: ", 0))
		sr.Deployer = singularity.NewDeployer(sr.Registry, drc)
	}
}
