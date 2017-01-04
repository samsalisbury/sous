package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousPlumbingStatus is the `sous plumbing status` object
type SousPlumbingStatus struct {
	config.DeployFilterFlags
	*sous.StatusPoller
}

func init() { PlumbingSubcommands["status"] = &SousPlumbingStatus{} }

// Help implements Command on SousPlumbingStatus
func (*SousPlumbingStatus) Help() string {
	return `reports the status of a given deployment`
}

// AddFlags implements cmdr.AddFlags on SousPlumbingStatus
func (sps *SousPlumbingStatus) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sps.DeployFilterFlags, ManifestFilterFlagsHelp)
}

// RegisterOn implements Registrant on SousPlumbingStatus
func (sps *SousPlumbingStatus) RegisterOn(psy Addable) {
	psy.Add(&sps.DeployFilterFlags)
}

// Execute implements cmdr.Executor on SousPlumbingStatus
func (sps *SousPlumbingStatus) Execute(args []string) cmdr.Result {
	sps.StatusPoller.Start()

	return cmdr.Success()
}
