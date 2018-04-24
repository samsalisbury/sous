package actions

import (
	"fmt"
	"io"
	"io/ioutil"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/yaml"
)

// ManifestSet is an Action for setting a manifest
type ManifestSet struct {
	sous.ManifestID
	InReader      io.Reader
	ResolveFilter *sous.ResolveFilter
	logging.LogSink
	User    sous.User
	Updater *restful.Updater
}

// Do implements the Action interface on ManifestSet
func (ms *ManifestSet) Do() error {
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

	if ms.ManifestID.Source.Repo != yml.Source.Repo {
		return fmt.Errorf("sous does not support changing source location, please use sous init")
	}

	_, err = (*ms.Updater).Update(&yml, nil)
	if err != nil {
		return err
	}

	return nil
}
