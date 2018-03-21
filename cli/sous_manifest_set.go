package cli

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/restful"
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

// Help implements part of the cmdr interfaces.
func (*SousManifestSet) Help() string { return sousManifestHelp }

// AddFlags implements part of the cmdr interfaces.
func (smg *SousManifestSet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, ManifestFilterFlagsHelp)
}

// Execute implements part of the cmdr interfaces.
func (smg *SousManifestSet) Execute(args []string) cmdr.Result {
	var up restful.Updater
	get, err := smg.SousGraph.GetManifestGet(smg.DeployFilterFlags, ioutil.Discard, func(u restful.Updater) {
		up = u
	})
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	set, err := smg.SousGraph.GetManifestSet(smg.DeployFilterFlags, up, os.Stdin)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := get.Do(); err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := set.Do(); err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	return cmdr.Success()
}
