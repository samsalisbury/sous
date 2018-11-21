package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestSousQueryGDM_dump(t *testing.T) {
	mkUniqDeployment := func(repo string) *sous.Deployment {
		return &sous.Deployment{SourceID: sous.MustNewSourceID(repo, "", "1")}
	}
	getOutput := func(formatFlag string) string {
		ds := sous.NewDeployments(
			mkUniqDeployment("repo1"),
		)
		sb := &SousQueryGDM{}
		gotBuf := &bytes.Buffer{}
		sb.Out = gotBuf

		sb.flags.format = formatFlag

		sb.dump(ds)
		return gotBuf.String()
	}

	var gotDefault, gotTable string

	t.Run("default", func(t *testing.T) {
		gotDefault = getOutput("")
		if !strings.Contains(gotDefault, "repo1") {
			t.Errorf("got output not containing repo1: %s", gotTable)
		}
	})

	t.Run("table", func(t *testing.T) {
		gotTable = getOutput("table")
		if !strings.Contains(gotTable, "repo1") {
			t.Errorf("got output not containing repo1: %s", gotTable)
		}
	})

	if gotDefault != gotTable {
		t.Errorf("default != table format:\ndefault:\n%s\n\ntable:\n%s",
			gotDefault, gotTable)
	}

	t.Run("json", func(t *testing.T) {
		gotJSON := getOutput("json")
		if !strings.Contains(gotJSON, "repo1") {
			t.Errorf("got output not containing repo1: %s", gotJSON)
		}
		d := sous.Deployment{}
		if err := json.Unmarshal([]byte(gotJSON), &d); err != nil {
			t.Errorf("invalid JSON: %s; output was:\n%s", err, gotJSON)
		}
	})
}
