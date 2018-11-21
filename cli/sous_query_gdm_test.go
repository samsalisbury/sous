package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/opentable/sous/cli/queries"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

func TestSousQueryGDM_dump(t *testing.T) {
	mkUniqDeployment := func(repo string) *sous.Deployment {
		return &sous.Deployment{SourceID: sous.MustNewSourceID(repo, "", "1")}
	}
	getOutputErr := func(t *testing.T, formatFlag string) (string, error) {
		t.Helper()
		ds := sous.NewDeployments(
			mkUniqDeployment("repo1"),
		)
		sb := &SousQueryGDM{}
		gotBuf := &bytes.Buffer{}
		sb.Out = gotBuf

		sb.flags.format = formatFlag

		err := sb.dump(ds)
		return gotBuf.String(), err
	}

	getOutput := func(t *testing.T, formatFlag string) string {
		t.Helper()
		o, err := getOutputErr(t, formatFlag)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}

	var gotDefault, gotTable string

	t.Run("default", func(t *testing.T) {
		gotDefault = getOutput(t, "")
		if !strings.Contains(gotDefault, "repo1") {
			t.Errorf("got output not containing repo1: %s", gotDefault)
		}
	})

	t.Run("table", func(t *testing.T) {
		gotTable = getOutput(t, "table")
		if !strings.Contains(gotTable, "repo1") {
			t.Errorf("got output not containing repo1: %s", gotTable)
		}
	})

	if gotDefault != gotTable {
		t.Errorf("default != table format:\ndefault:\n%s\n\ntable:\n%s",
			gotDefault, gotTable)
	}

	t.Run("json", func(t *testing.T) {
		gotJSON := getOutput(t, "json")
		if !strings.Contains(gotJSON, "repo1") {
			t.Errorf("got output not containing repo1: %s", gotJSON)
		}
		d := sous.Deployment{}
		if err := json.Unmarshal([]byte(gotJSON), &d); err != nil {
			t.Errorf("invalid JSON: %s; output was:\n%s", err, gotJSON)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		got, gotErr := getOutputErr(t, "invalid")
		wantErr := `output format "invalid" not valid`
		if gotErr == nil {
			t.Fatalf("got nil; want error containing %q", wantErr)
		}
		if !strings.Contains(gotErr.(error).Error(), wantErr) {
			t.Fatalf("got error %q; want error containing %q", gotErr, wantErr)
		}
		if got != gotDefault {
			t.Fatalf("output for error not the same as default:\ngot:\n%s\nwant:\n%s",
				got, gotDefault)
		}
	})
}

func TestSousQueryGDM_Execute(t *testing.T) {
	assertExitCode := func(format, filters string, wantExitCode int) {
		t.Run(fmt.Sprintf("-format=%s -filters='%s'", format, filters), func(t *testing.T) {
			sb := SousQueryGDM{
				DeploymentQuery: queries.Deployment{
					StateManager: sous.NewDummyStateManager(),
					ArtifactQuery: queries.ArtifactQuery{
						Client: &restful.DummyHTTPClient{},
					},
				},
				flags: struct {
					filters string
					format  string
				}{
					filters: filters,
					format:  format,
				},
				Out: ioutil.Discard,
			}
			got := sb.Execute(nil).ExitCode()
			if got != wantExitCode {
				t.Errorf("got exit code %d; want %d", got, wantExitCode)
			}
		})
	}

	const success = 0
	const usageErr = 64

	assertExitCode("", "", success)
	assertExitCode("json", "hasimage=true", success)
	assertExitCode("table", "", success)

	assertExitCode("table", "invalid", usageErr)
	assertExitCode("", "invalid", usageErr)
	assertExitCode("invalid", "", usageErr)
	assertExitCode("invalid", "invalid", usageErr)
}
