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
	graph.TargetManifestID
	graph.HTTPClient
	graph.InReader
	ResolveFilter graph.RefinedResolveFilter `inject:"optional"`
	graph.LogSink
	User sous.User
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

func (smg *SousManifestSet) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&smg.DeployFilterFlags)
}

func (smg *SousManifestSet) Execute(args []string) cmdr.Result {
	mani := sous.Manifest{}
	up, err := smg.HTTPClient.Retrieve("/manifest", smg.TargetManifestID.QueryMap(), &mani, nil)

	if err != nil {
		return EnsureErrorResult(errors.Errorf("No manifest matched by %v yet. See `sous init` (%v)", smg.ResolveFilter, err))
	}

	yml := sous.Manifest{}
	bytes, err := ioutil.ReadAll(smg.InReader)
	if err != nil {
		return EnsureErrorResult(err)
	}
	err = yaml.Unmarshal(bytes, &yml)
	if err != nil {
		return EnsureErrorResult(err)
	}

	messages.ReportLogFieldsMessage("Manifest in Execute", logging.ExtraDebug1Level, smg.LogSink, yml)

	_, err = up.Update(&yml, nil)
	if err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}
