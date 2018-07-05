package actions

import (
	"fmt"
	"strings"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/shell"
)

//AddArtifact struct for add artifact action.
type AddArtifact struct {
	Repo        string
	Offset      string
	Tag         string
	DockerImage string
	LocalShell  shell.Shell
	LogSink     logging.LogSink
	User        sous.User
	Inserter    sous.ClientInserter
	*config.Config
}

//Do executes the action for add artifact.
func (a *AddArtifact) Do() error {

	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Executing add artifact Repo: %s, Offset: %s, Tag: %s, DockerImage: %s", a.Repo, a.Offset, a.Tag, a.DockerImage), logging.ExtraDebug1Level, a.LogSink)

	versionName := fmt.Sprintf("%s:%s", a.DockerImage, a.Tag)
	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Attempting to get Digest for: %s", versionName), logging.ExtraDebug1Level, a.LogSink)

	output, err := a.LocalShell.Stdout("docker", "inspect", "--format='{{index .RepoDigests 0}}'", versionName)
	if err != nil {
		return err
	}
	digestName := strings.Replace(strings.TrimSpace(output), "'", "", -1)

	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Digest for: %s is %s", versionName, digestName), logging.ExtraDebug1Level, a.LogSink)

	return a.UploadArtifact(versionName, digestName)
}

//UploadArtifact add artifact to sous servers
func (a *AddArtifact) UploadArtifact(versionName, digestName string) error {

	art := sous.BuildArtifact{
		VersionName:     versionName,
		DigestReference: digestName,
		Qualities: []sous.Quality{{
			Name: "added artifact",
			Kind: "advisory",
		}},
	}

	sid := sous.MakeSourceID(a.Repo, a.Offset, a.Tag)

	return a.Inserter.Insert(sid, art)
}
