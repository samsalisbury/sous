package actions

import (
	"fmt"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
)

//AddArtifact struct for normalize GDM action.
type AddArtifact struct {
	Repo       string
	Cluster    string
	HTTPClient restful.HTTPClient
	LogSink    logging.LogSink
	User       sous.User
	*config.Config
}

//Do executes the action for plumb normalize GDM.
func (a *AddArtifact) Do() error {

	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Executing add artifact %s, %s", a.Repo, a.Cluster), logging.ExtraDebug1Level, a.LogSink)

	return nil
}
