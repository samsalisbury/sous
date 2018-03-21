package actions

import (
	"bytes"
	"os"
	"testing"

	"github.com/nyarly/spies"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful/restfultest"
	"github.com/opentable/sous/util/yaml"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestSet(t *testing.T) {
	project1 := sous.SourceLocation{Repo: "github.com/user/project"}

	cl, control := restfultest.NewHTTPClientSpy()
	mid := sous.ManifestID{
		Source: sous.SourceLocation{
			Repo: project1.Repo,
		},
	}

	mani := sous.ManifestFixture("simple")

	mani.Flavor = "vanilla"
	yml, err := yaml.Marshal(mani)
	require.NoError(t, err)
	in := bytes.NewBuffer(yml)

	sms := &ManifestSet{
		ManifestID: mid,

		HTTPClient: cl,

		InReader: in,
		LogSink:  logging.NewLogSet(semv.MustParse("0.0.0"), "", "", os.Stderr),
	}

	updater, upctl := restfultest.NewUpdateSpy()
	control.MatchMethod(
		"Retrieve",
		spies.Once(),
		sous.ManifestFixture("simple"), updater, nil,
	)
	control.Any(
		"Retrieve",
		sous.ManifestFixture("simple"), restfultest.DummyUpdater(), nil,
	)
	upctl.Any(
		"Update",
		nil,
	)

	err = sms.Do()
	assert.NoError(t, err)

	if assert.Len(t, control.Calls(), 1) {
		args := control.Calls()[0].PassedArgs()
		assert.Regexp(t, "/manifest", args.String(0))
	}
	if assert.Len(t, upctl.Calls(), 1) {
		args := upctl.Calls()[0].PassedArgs()
		assert.Equal(t, args.Get(0).(*sous.Manifest).Flavor, "vanilla")
	}
}
