package actions

import (
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
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
	LocalShell  shell.Shell
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
	output, err := a.LocalShell.Stdout("docker", "inspect", "--format='{{index .RepoDigests 0}}'", versionName)
	if err != nil {
		return err
	}

	digestName := strings.Trim(output, " \n\t\r")
	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Digest for: %s is %s", versionName, digestName), logging.ExtraDebug1Level, a.LogSink)

	return nil
}

func (a *AddArtifact) UploadArtifact(versionName, digestName string) error {

	art := sous.BuildArtifact{
		VersionName:     versionName,
		DigestReference: digestName,
		Qualities:       []sous.Quality{{"otpl_tag", "advisory"}},
	}

	spew.Printf("%v", art)
	/*
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.Encode(art)
		req, err := http.NewRequest("PUT", "", buf)
		if err != nil {
			return fmt.Errorf("error building request %s", err.Error())
		}

		query := fmt.Sprintf("repo=%s&offset=%s&version=", a.Repo, a.Offset, a.Tag)

		q, err := url.ParseQuery(query)
		if err != nil {
			return fmt.Errorf("failed to parse query %s: %s", query, err.Error())
		}
		/*
			ins, ctrl := sous.NewInserterSpy()

			pah := &PUTArtifactHandler{
				Request:     req,
				QueryValues: restful.QueryValues{Values: q},
				Inserter:    ins,
			}

			_, status := pah.Exchange()

			assert.Equal(t, 200, status, "status")

			require.Len(t, ctrl.CallsTo("Insert"), 1)
			ic := ctrl.CallsTo("Insert")[0]
			inSid := ic.PassedArgs().Get(0).(sous.SourceID)
			inBA := ic.PassedArgs().Get(1).(sous.BuildArtifact)

			assert.Equal(t, "github.com/opentable/test", inSid.Location.Repo, "source id repo")
			assert.Equal(t, art.DigestReference, inBA.DigestReference, "build artifact digest name")
			assert.Equal(t, art.VersionName, inBA.VersionName, "build artifact version name")
	*/

	return nil
}
