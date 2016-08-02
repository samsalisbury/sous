package cli

import (
	"flag"
	"log"
	"os"

	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousRectify is the injectable command object used for `sous rectify`
type (
	SousRectify struct {
		Config       LocalSousConfig
		DockerClient LocalDockerClient
		Deployer     sous.Deployer
		Registry     sous.Registry
		GDM          CurrentGDM
		flags        rectifyFlags
	}

	rectifyFlags struct {
		dryrun,
		repo, offset, cluster string
		all bool
	}
)

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
	fs.StringVar(&sr.flags.dryrun, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
	fs.StringVar(&sr.flags.repo, "repo", "",
		"consider only the repo `repository` for rectification")
	fs.StringVar(&sr.flags.offset, "offset", "",
		"consider only the offset `path` for rectification")
	fs.StringVar(&sr.flags.cluster, "cluster", "",
		"consider only the cluster `name` for rectification")
	fs.BoolVar(&sr.flags.all, "all", false,
		"actually do a full-tree recitification")
}

// Execute fulfils the cmdr.Executor interface
func (sr *SousRectify) Execute(args []string) cmdr.Result {

	sr.resolveDryRunFlag(sr.flags.dryrun)

	predicate := sr.flags.buildPredicate()

	r := sous.NewResolver(sr.Deployer, sr.Registry)

	// If predicate is still nil, that means resolve all. See Deployments.Filter.
	if err := r.ResolveFilteredDeployments(*sr.GDM.State, predicate); err != nil {
		return EnsureErrorResult(err)
	}

	return Success()
}

func (f rectifyFlags) buildPredicate() sous.DeploymentPredicate {
	var preds []sous.DeploymentPredicate

	if f.all {
		return func(*sous.Deployment) bool { return true }
	}

	if f.repo != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.RepoURL == sous.RepoURL(f.repo)
		})
	}

	if f.offset != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.SourceID.RepoOffset == sous.RepoOffset(f.offset)
		})
	}

	if f.cluster != "" {
		preds = append(preds, func(d *sous.Deployment) bool {
			return d.ClusterNickname == f.cluster
		})
	}

	// These aren't strictly necessary, but an easy optimization
	switch len(preds) {
	case 0:
		return nil
	case 1:
		return preds[0]
	default:
		return func(d *sous.Deployment) bool {
			for _, f := range preds {
				if !f(d) { // AND(preds...)
					return false
				}
			}
			return true
		}
	}
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
