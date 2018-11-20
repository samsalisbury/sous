package actions

import (
	"fmt"
	"os"
	"strings"

	"github.com/opentable/sous/cli/queries"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/pkg/errors"
)

//GetArtifact struct for add artifact action.
type GetArtifact struct {
	Query             queries.ArtifactQuery
	Repo, Offset, Tag string
	LogSink           logging.LogSink
	BuildArtifact     sous.BuildArtifact
}

//Do executes the action for add artifact.
func (a *GetArtifact) Do() error {

	messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Executing get artifact Repo: %s, Offset: %s, Version: %s", a.Repo, a.Offset, a.Tag), logging.ExtraDebug1Level, a.LogSink)

	sid, err := sous.NewSourceID(a.Repo, a.Offset, a.Tag)
	if err != nil {
		return fmt.Errorf("source ID not valid: %s", err)
	}

	ba, err := a.Query.ByID(sid)
	if err != nil {
		return fmt.Errorf("failed to retrieve artifact: %s", err)
	}

	if ba == nil {
		return errors.Errorf("\nNo artifact matched:\n\n\tPlease check your repo, version, and offset.  Items are case sensitive.  Use the following command to verify values sous expects.\n\n\tsous query gdm\n\nError returned: %s", err)
	}
	messages.ReportLogFieldsMessage("GetArtifact.Execute Retrieved BuildArtifact",
		logging.ExtraDebug1Level, a.LogSink, ba)

	a.BuildArtifact = *ba

	fmt.Fprintf(os.Stderr, "name: %s\ndigest: %s\ntype: %s\n", ba.VersionName, ba.DigestReference, ba.Type)

	return nil
}

// ArtifactExists returns (true, nil) if the artifact exists,
// (false, nil) it the artifact does not exist and (undefined, error)
// if the check is not successful.
func (a *GetArtifact) ArtifactExists() (bool, error) {
	err := a.Do()
	if err == nil {
		return true, nil // fmt.Errorf("artifact already registered")
	}
	if strings.Contains(err.Error(), "404 Not Found") {
		return false, nil
	}
	return false, err
}
