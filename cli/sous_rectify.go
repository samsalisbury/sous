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
	Deployer     sous.Deployer
	Registry     sous.Registry
	GDM          CurrentGDM
	flags        struct {
		dryrun,
		manifest string
	}
}

func init() { TopLevelCommands["rectify"] = &SousRectify{} }

const sousRectifyHelp = `
force Sous to make the deployment match the contents of the local state directory

usage: sous rectify

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
	fs.StringVar(&sr.flags.manifest, "manifest", "",
		"consider only the named manifest for rectification")
}

// Execute fulfils the cmdr.Executor interface
func (sr *SousRectify) Execute(args []string) cmdr.Result {

	sr.resolveDryRunFlag(sr.flags.dryrun)

	var predicate sous.DeploymentPredicate
	if sr.flags.manifest != "" {
		predicate = func(d *sous.Deployment) bool {
			return d.SourceVersion.RepoURL == sous.RepoURL(sr.flags.manifest)
		}
	}

	r := sous.NewResolver(sr.Deployer, sr.Registry)

	// If predicate is still nil, that means resolve all. See Deployments.Filter.
	if err := r.ResolveFilteredDeployments(*sr.GDM.State, predicate); err != nil {
		return EnsureErrorResult(err)
	}

	return Success()
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
