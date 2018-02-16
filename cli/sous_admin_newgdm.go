package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/firsterr"
)

// SousNewGDM creates a new GDM at the configured StateLocation.
type SousNewGDM struct {
	WD          graph.LocalWorkDirShell
	StateWriter graph.StateWriter
	Config      graph.LocalSousConfig
	User        sous.User
	flags       struct {
		CheckoutDir,
		Clusters,
		SingularityURL,
		DockerRegURL string
	}
}

// NOTE: For now this only creates "test" GDMs which have FS based Git remotes.
func init() { TopLevelCommands["new-test-gdm"] = &SousNewGDM{} }

const sousNewGDMHelp = `initialise a new test sous GDM

usage: sous newgdm -dir dir -clusters a,b,c -singularities a[,b[,c]] -docker-reg somereg
`

// Help returns the help string for this command
func (si *SousNewGDM) Help() string { return sousNewGDMHelp }

// AddFlags adds the flags for sous init.
func (si *SousNewGDM) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&si.flags.CheckoutDir, "dir", "", "dir to create gdm")
	fs.StringVar(&si.flags.Clusters, "clusters", "", "comma-separated cluster names")
	fs.StringVar(&si.flags.SingularityURL, "singularity", "", "singularity URL")
	fs.StringVar(&si.flags.DockerRegURL, "docker-reg", "", "docker registry URL")
}

// RegisterOn adds flag sets for sous init to the dependency injector.
func (si *SousNewGDM) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&config.DeployFilterFlags{})
}

// Execute fulfils the cmdr.Executor interface
func (si *SousNewGDM) Execute(args []string) cmdr.Result {

	requireFlag := func(name, val string) error {
		if val != "" {
			return nil
		}
		return fmt.Errorf("must specify flag -%s", name)
	}

	flagErr := firsterr.Set(
		func(e *error) { *e = requireFlag("dir", si.flags.CheckoutDir) },
		func(e *error) { *e = requireFlag("clusters", si.flags.Clusters) },
		func(e *error) { *e = requireFlag("singularity", si.flags.SingularityURL) },
		func(e *error) { *e = requireFlag("docker-reg", si.flags.DockerRegURL) },
	)
	if flagErr != nil {
		return cmdr.EnsureErrorResult(flagErr)
	}

	dsm := storage.NewDiskStateManager(si.flags.CheckoutDir)

	state := sous.NewState()
	state.Defs.DockerRepo = si.flags.DockerRegURL
	state.Defs.Clusters = sous.Clusters{}
	for _, c := range strings.Split(si.flags.Clusters, ",") {
		state.Defs.Clusters[c] = &sous.Cluster{
			Name:              c,
			Kind:              "singularity",
			BaseURL:           si.flags.SingularityURL,
			AllowedAdvisories: sous.VeryPermissiveAdvisories().Strings(),
		}
	}

	if err := dsm.WriteState(state, si.User); err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	return cmdr.Successf("State initialized at %s", si.flags.CheckoutDir)
}
