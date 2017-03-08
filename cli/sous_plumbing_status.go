package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	state, err := sps.StatusPoller.Wait(ctx)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	if state != sous.ResolveComplete {
		return cmdr.EnsureErrorResult(fmt.Errorf("failed"))
	}

	return cmdr.Success()
}
