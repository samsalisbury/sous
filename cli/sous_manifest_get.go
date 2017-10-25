package cli

import (
	"flag"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
)

type SousManifestGet struct {
	config.DeployFilterFlags
	ResolveFilter *graph.RefinedResolveFilter
	graph.TargetManifestID
	graph.HTTPClient
	graph.LogSink
	graph.OutWriter
}

func init() { ManifestSubcommands["get"] = &SousManifestGet{} }

const sousManifestGetHelp = `query deployment manifest`

func (*SousManifestGet) Help() string { return sousManifestHelp }

func (smg *SousManifestGet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, ManifestFilterFlagsHelp)
}

func (smg *SousManifestGet) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&smg.DeployFilterFlags)
}

func (smg *SousManifestGet) Execute(args []string) cmdr.Result {
	mani := sous.Manifest{}
	_, err := smg.HTTPClient.Retrieve("./manifest", smg.TargetManifestID.QueryMap(), &mani, nil)

	if err != nil {
		return EnsureErrorResult(errors.Errorf("No manifest matched by %v yet. See `sous init` (%v)", smg.ResolveFilter, err))
	}
	smg.Vomitf(spew.Sdump(mani))

	yml, err := yaml.Marshal(mani)
	if err != nil {
		return EnsureErrorResult(err)
	}
	smg.OutWriter.Write(yml)
	return cmdr.Success()
}
