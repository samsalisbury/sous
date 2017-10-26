package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousPlumbingStatus is the `sous plumbing status` object.
type SousPlumbingStatus struct {
	SousGraph *graph.SousGraph
	Config    graph.LocalSousConfig

	DeployFilterFlags config.DeployFilterFlags
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

// Execute implements cmdr.Executor on SousPlumbingStatus.
func (sps *SousPlumbingStatus) Execute(args []string) cmdr.Result {
	if sps.Config.Server == "" {
		return cmdr.UsageErrorf("Please configure a server using 'sous config Server <url>'")
	}

	poll := sps.SousGraph.GetPollStatus("none", sps.DeployFilterFlags)
	if err := poll.Do(); err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	return cmdr.Success()

}
