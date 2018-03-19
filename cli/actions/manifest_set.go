package actions

import (
	"io/ioutil"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// ManifestSet is an Action for setting a manifest
type ManifestSet struct {
	config.DeployFilterFlags `inject:"optional"`
	graph.TargetManifestID
	graph.HTTPClient
	graph.InReader
	ResolveFilter graph.RefinedResolveFilter `inject:"optional"`
	graph.LogSink
	User sous.User
}

// Do implements the Action interface on ManifestSet
func (ms *ManifestSet) Do() {
	mani := sous.Manifest{}
	up, err := ms.HTTPClient.Retrieve("/manifest", ms.TargetManifestID.QueryMap(), &mani, nil)

	if err != nil {
		return EnsureErrorResult(errors.Errorf("No manifest matched by %v yet. See `sous init` (%v)", ms.ResolveFilter, err))
	}

	yml := sous.Manifest{}
	bytes, err := ioutil.ReadAll(ms.InReader)
	if err != nil {
		return EnsureErrorResult(err)
	}
	err = yaml.Unmarshal(bytes, &yml)
	if err != nil {
		return EnsureErrorResult(err)
	}

	messages.ReportLogFieldsMessage("Manifest in Execute", logging.ExtraDebug1Level, ms.LogSink, yml)

	_, err = up.Update(&yml, nil)
	if err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}
