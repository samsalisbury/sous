package actions

import (
	"fmt"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/shell"
	"github.com/pkg/errors"
)

//GetArtifact struct for add artifact action.
type GetArtifact struct {
	Repo          string
	Offset        string
	Tag           string
	LocalShell    shell.Shell
	LogSink       logging.LogSink
	User          sous.User
	BuildArtifact sous.BuildArtifact
	HTTPClient    restful.HTTPClient
	*config.Config
}

//Do executes the action for add artifact.
func (a *GetArtifact) Do() error {

	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Executing get artifact Repo: %s, Offset: %s, Version: %s", a.Repo, a.Offset, a.Tag), logging.ExtraDebug1Level, a.LogSink)

	ba := sous.BuildArtifact{}
	artifactQuery := map[string]string{}
	artifactQuery["repo"] = a.Repo
	artifactQuery["offset"] = a.Offset
	artifactQuery["version"] = a.Tag

	_, err := a.HTTPClient.Retrieve("./artifact", artifactQuery, &ba, a.User.HTTPHeaders())
	if err != nil {
		return errors.Errorf("\nFailed to retrieve artrifact:\n\n\tPlease check your repo, version, and offset.  Items are case sensitive.  Use the following command to verify values sous expects.\n\n\tsous query gdm\n\nError returned: %s", err)
	}
	messages.ReportLogFieldsMessage("GetArtifact.Execute Retrieved BuildArtifact",
		logging.ExtraDebug1Level, a.LogSink, ba)

	a.BuildArtifact = ba

	return nil
}
