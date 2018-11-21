package actions

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/restful/restfultest"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestGet(t *testing.T) {
	assertError := func(manifestReturned interface{}, gotHTTPErr, wantErr bool) {
		var httpErr error
		if gotHTTPErr {
			httpErr = fmt.Errorf("an error")
		}
		out := &bytes.Buffer{}
		project1 := sous.SourceLocation{Repo: "github.com/user/project"}

		var up restful.Updater
		cl, control := restfultest.NewHTTPClientSpy()
		smg := &ManifestGet{
			TargetManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: project1.Repo,
				},
				Flavor: "chocolate",
			},
			HTTPClient: cl,

			OutWriter:      out,
			LogSink:        logging.NewLogSet(semv.MustParse("0.0.0"), "", "", os.Stderr),
			UpdaterCapture: &up,
		}

		control.Any(
			"Retrieve",
			manifestReturned, restfultest.DummyUpdater(), httpErr,
		)

		err := smg.Do()
		if wantErr {
			assert.Error(t, err)
			return
		}
		require.NoError(t, err)

		if assert.Len(t, control.Calls(), 1) {
			assert.Regexp(t, "/manifest", control.Calls()[0].PassedArgs().String(0))
			params := control.Calls()[0].PassedArgs().Get(1).(map[string]string)
			assert.Contains(t, params, "repo")
			assert.Contains(t, params, "flavor")
			assert.Equal(t, params["flavor"], "chocolate")
		}

		assert.Regexp(t, "github", out.String())
	}

	assertError(sous.ManifestFixture("simple"), false, false)
	assertError(sous.ManifestFixture("simple"), true, true)

}
