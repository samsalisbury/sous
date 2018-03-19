package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

type SousManifestSet struct {
	config.DeployFilterFlags `inject:"optional"`
	SousGraph                *graph.SousGraph
}

func init() { ManifestSubcommands["set"] = &SousManifestSet{} }

const sousManifestSetHelp = `replace a deployment manifest

do note: this does *replace* the manifest;
there's some validation, but you can make drastic changes easily
`

func (*SousManifestSet) Help() string { return sousManifestHelp }

func (smg *SousManifestSet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, ManifestFilterFlagsHelp)
}

func (smg *SousManifestSet) Execute(args []string) cmdr.Result {
	set, err := smg.SousGraph.GetManifestSet(smg.DeployFilterFlags)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := set.Do(); err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	return cmdr.Success()
}
