package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousNewGDM creates a new GDM at the configured StateLocation.
type SousNewGDM struct {
	WD          graph.LocalWorkDirShell
	StateWriter graph.StateWriter
	Config      graph.LocalSousConfig
	User        sous.User
	flags       struct {
		CheckoutDir string
		RemoteDir   string
	}
}

// NOTE: For now this only creates "test" GDMs which have FS based Git remotes.
func init() { TopLevelCommands["new-test-gdm"] = &SousNewGDM{} }

const sousNewGDMHelp = `initialise a new test sous GDM

usage: sous newgdm
`

// Help returns the help string for this command
func (si *SousNewGDM) Help() string { return sousNewGDMHelp }

// AddFlags adds the flags for sous init.
func (si *SousNewGDM) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&si.flags.CheckoutDir, "checkout-dir", "", "dir to create live checkout (defaults to config.StateLocation)")
	fs.StringVar(&si.flags.RemoteDir, "remote-dir", "", "dir to create git remote")
}

// RegisterOn adds flag sets for sous init to the dependency injector.
func (si *SousNewGDM) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&config.DeployFilterFlags{})
}

// Execute fulfills the cmdr.Executor interface
func (si *SousNewGDM) Execute(args []string) cmdr.Result {

	checkoutDir := func() string {
		if si.flags.CheckoutDir != "" {
			return si.flags.CheckoutDir
		}
		return si.Config.StateLocation
	}()

	dsm := storage.NewDiskStateManager(checkoutDir)

	state := sous.NewState()
	state.Defs.Clusters = sous.Clusters{
		"cluster1": &sous.Cluster{
			Name:              "cluster1",
			Kind:              "singularity",
			BaseURL:           "??",
			AllowedAdvisories: sous.VeryPermissiveAdvisories().Strings(),
		},
		"cluster2": &sous.Cluster{
			Name:              "cluster2",
			Kind:              "singularity",
			BaseURL:           "??",
			AllowedAdvisories: sous.VeryPermissiveAdvisories().Strings(),
		},
	}

	if err := dsm.WriteState(state, si.User); err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	return cmdr.Successf("State initialized at %s", checkoutDir)
}
