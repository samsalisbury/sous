package actions

import (
	"fmt"
	"strings"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/shell"
)

//AddArtifact struct for normalize GDM action.
type AddArtifact struct {
	Repo        string
	Cluster     string
	Offset      string
	Tag         string
	DockerImage string
	SourceShell shell.Shell
	HTTPClient  restful.HTTPClient
	LogSink     logging.LogSink
	User        sous.User
	*config.Config
}

//Do executes the action for plumb normalize GDM.
func (a *AddArtifact) Do() error {

	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Executing add artifact Repo: %s, Cluster: %s, Offset: %s, Tag: %s, DockerImage: %s", a.Repo, a.Cluster, a.Offset, a.Tag, a.DockerImage), logging.ExtraDebug1Level, a.LogSink)

	versionName := fmt.Sprintf("%s:%s", a.DockerImage, a.Tag)
	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Attempting to get Digest for: %s", versionName), logging.ExtraDebug1Level, a.LogSink)

	//Now figure out the digest
	output, err := a.SourceShell.Stdout("docker", "inspect", "--format='{{index .RepoDigests 0}}'", versionName)
	if err != nil {
		return err
	}

	digestName := strings.Trim(output, " \n\t\r")

	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Digest for: %s is %s", versionName, digestName), logging.ExtraDebug1Level, a.LogSink)

	return nil
}
