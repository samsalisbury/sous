package cli

import (
	"flag"
	"io/ioutil"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
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

	if err := set.Do(); err {
		return cmdr.EnsureErrorResult(err)
	}

	return cmdr.Success()
}
