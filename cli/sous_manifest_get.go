package cli

import (
	"flag"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/restful"
)

// SousManifestGet defines the `sous manifest get` command.
type SousManifestGet struct {
	config.DeployFilterFlags `inject:"optional"`
	SousGraph                *graph.SousGraph
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
	mg, err := smg.SousGraph.GetManifestGet(smg.DeployFilterFlags, os.Stdout, func(restful.Updater) {})
	if err != nil {
		return EnsureErrorResult(err)
	}

	if err := mg.Do(); err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}
