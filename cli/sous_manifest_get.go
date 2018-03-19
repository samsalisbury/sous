package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

type SousManifestGet struct {
	config.DeployFilterFlags `inject:"optional"`

	ResolveFilter    *graph.RefinedResolveFilter `inject:"optional"`
	TargetManifestID graph.TargetManifestID
	HTTPClient       graph.HTTPClient
	LogSink          graph.LogSink
	OutWriter        graph.OutWriter

	SousGraph *graph.SousGraph
}

func init() { ManifestSubcommands["get"] = &SousManifestGet{} }

const sousManifestGetHelp = `query deployment manifest`

// Help implements Command on SousManifestGet.
func (*SousManifestGet) Help() string { return sousManifestHelp }

// AddFlags implements AddFlagger on SousManifestGet.
func (smg *SousManifestGet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, ManifestFilterFlagsHelp)
}

// Execute implements Executor on SousManifestGet.
func (smg *SousManifestGet) Execute(args []string) cmdr.Result {
	mg, err := smg.SousGraph.GetManifestGet(smg.DeployFilterFlags)
	if err != nil {
		return EnsureErrorResult(err)
	}

	if err := mg.Do(); err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}
