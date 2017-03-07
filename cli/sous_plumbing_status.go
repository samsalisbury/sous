package cli

import (
	"flag"
	"fmt"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousPlumbingStatus is the `sous plumbing status` object.
type SousPlumbingStatus struct {
	DeployFilterFlags config.DeployFilterFlags
	StatusPoller      *sous.StatusPoller
}

func init() { PlumbingSubcommands["status"] = &SousPlumbingStatus{} }

// Help implements Command on SousPlumbingStatus.
func (*SousPlumbingStatus) Help() string {
	return `reports the status of a given deployment`
}

// AddFlags implements cmdr.AddFlags on SousPlumbingStatus.
func (sps *SousPlumbingStatus) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sps.DeployFilterFlags, DeployFilterFlagsHelp)
}

// RegisterOn implements Registrant on SousPlumbingStatus.
func (sps *SousPlumbingStatus) RegisterOn(psy Addable) {
	psy.Add(&sps.DeployFilterFlags)
}

// Execute implements cmdr.Executor on SousPlumbingStatus.
func (sps *SousPlumbingStatus) Execute(args []string) cmdr.Result {

	if sps.StatusPoller == nil {
		return cmdr.UsageErrorf("Please configure a server using 'sous config Server <url>'")
	}

	state, err := sps.StatusPoller.Start()
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	if state != sous.ResolveComplete {
		return cmdr.EnsureErrorResult(fmt.Errorf("failed"))
	}

	return cmdr.Success()
}
