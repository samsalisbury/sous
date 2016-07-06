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
type SousRectify struct {
	Config       LocalSousConfig
	DockerClient LocalDockerClient
	Builder      sous.Registry
	Deployer     sous.Deployer
	Registry     sous.Registry
	flags        struct {
		dryrun,
		manifest string
	}
}

func init() { TopLevelCommands["rectify"] = &SousRectify{} }

const sousRectifyHelp = `
force Sous to make the deployment match the contents of a state directory

usage: sous rectify <dir>

Note: by default this command will query a live docker registry and make
changes to live Mesos schedulers
`

// Help returns the help string
func (*SousRectify) Help() string { return sousRectifyHelp }

// AddFlags adds flags for sous rectify
func (sr *SousRectify) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&sr.flags.dryrun, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
	fs.StringVar(&sr.flags.manifest, "manifest", "",
		"consider only the named manifest for rectification")
}

// Execute fulfils the cmdr.Executor interface
func (sr *SousRectify) Execute(args []string) cmdr.Result {
	if len(args) < 1 {
		return UsageErrorf("sous rectify requires a directory to load the intended deployment from")
	}
	dir := args[0]

	sr.resolveDryRunFlag(sr.flags.dryrun)

	var predicate sous.DeploymentPredicate
	if sr.flags.manifest != "" {
		predicate = func(d *sous.Deployment) bool {
			return d.SourceVersion.RepoURL == sous.RepoURL(sr.flags.manifest)
		}
	}

	intendedState, err := sous.LoadState(dir)
	if err != nil {
		return EnsureErrorResult(err)
	}

	r := sous.NewResolver(sr.Deployer, sr.Registry, intendedState)

	// If predicate is still nil, that means resolve all. See Deployments.Filter.
	if err := r.ResolveFilteredDeployments(predicate); err != nil {
		return EnsureErrorResult(err)
	}

	return Success()
}

func (sr *SousRectify) resolveDryRunFlag(dryrun string) {
	if dryrun == "both" || dryrun == "registry" {
		sr.Builder = singularity.NewDummyRegistry()
	}
	if dryrun == "both" || dryrun == "scheduler" {
		drc := singularity.NewDummyRectificationClient(sr.Builder)
		drc.SetLogger(log.New(os.Stdout, "rectify: ", 0))
		sr.Deployer = drc
	}
}
