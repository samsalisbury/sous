package actions

import (
	"io"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
)

// A ManifestGet is an Action that fetches a manifest.
type ManifestGet struct {
	ResolveFilter    *sous.ResolveFilter
	TargetManifestID sous.ManifestID
	HTTPClient       restful.HTTPClient
	LogSink          logging.LogSink
	OutWriter        io.Writer
	UpdaterCapture   *restful.Updater
}

// Do implements Action on ManifestGet.
func (mg *ManifestGet) Do() error {
	mani := sous.Manifest{}
	up, err := mg.HTTPClient.Retrieve("./manifest", mg.TargetManifestID.QueryMap(), &mani, nil)

	if err != nil {
		return errors.Errorf("No manifest matched by %v yet. See `sous init` (%v)", mg.ResolveFilter, err)
	}
	(*mg.UpdaterCapture) = up

	messages.ReportLogFieldsMessage("Sous manifest in Execute", logging.ExtraDebug1Level, mg.LogSink, mani.ID())

	yml, err := yaml.Marshal(mani)
	if err != nil {
		return err
	}
	mg.OutWriter.Write(yml)
	return nil
}
