package actions

import (
	"io"
	"io/ioutil"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// ManifestSet is an Action for setting a manifest
type ManifestSet struct {
	config.DeployFilterFlags `inject:"optional"`
	sous.ManifestID
	restful.HTTPClient
	InReader      io.Reader
	ResolveFilter sous.ResolveFilter
	logging.LogSink
	User sous.User
}

// Do implements the Action interface on ManifestSet
func (ms *ManifestSet) Do() error {
	mani := sous.Manifest{}
	up, err := ms.HTTPClient.Retrieve("/manifest", ms.ManifestID.QueryMap(), &mani, nil)

	if err != nil {
		return errors.Errorf("No manifest matched by %v yet. See `sous init` (%v)", ms.ResolveFilter, err)
	}

	yml := sous.Manifest{}
	bytes, err := ioutil.ReadAll(ms.InReader)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bytes, &yml)
	if err != nil {
		return err
	}

	messages.ReportLogFieldsMessage("Manifest in Execute", logging.ExtraDebug1Level, ms.LogSink, yml)

	_, err = up.Update(&yml, nil)
	if err != nil {
		return err
	}

	return nil
}
