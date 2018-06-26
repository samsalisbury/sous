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
	Repo        string
	Cluster     string
	Offset      string
	Tag         string
	DockerImage string
	HTTPClient  restful.HTTPClient
	LogSink     logging.LogSink
	User        sous.User
	*config.Config
}

//Do executes the action for plumb normalize GDM.
func (a *AddArtifact) Do() error {

	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Executing add artifact Repo: %s, Cluster: %s, Offset: %s, Tag: %s, DockerImage: %s", a.Repo, a.Cluster, a.Offset, a.Tag, a.DockerImage), logging.ExtraDebug1Level, a.LogSink)

	return nil
}
