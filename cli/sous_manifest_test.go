package cli

import (
	"bytes"
	"flag"
	"os"
	"testing"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful/restfultest"
	"github.com/opentable/sous/util/spies"
	"github.com/opentable/sous/util/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestGetArgs(t *testing.T) {
	fs := flag.NewFlagSet("test-for-manifest-get", flag.ContinueOnError)
	smg := &SousManifestGet{}
	smg.AddFlags(fs)
	fs.Parse([]string{"-repo", "github.com/example/test", "-flavor", "winning"})

	assert.Equal(t, "github.com/example/test", smg.DeployFilterFlags.Repo)
	assert.Equal(t, "winning", smg.DeployFilterFlags.Flavor)
}

func TestManifestSetArgs(t *testing.T) {
	fs := flag.NewFlagSet("test-for-manifest-get", flag.ContinueOnError)
	smg := &SousManifestSet{}
	smg.AddFlags(fs)
	fs.Parse([]string{"-repo", "github.com/example/test", "-flavor", "winning"})

	assert.Equal(t, "github.com/example/test", smg.DeployFilterFlags.Repo)
	assert.Equal(t, "winning", smg.DeployFilterFlags.Flavor)
}

func TestManifestGet(t *testing.T) {
	out := &bytes.Buffer{}

	cl, control := restfultest.NewHTTPClientSpy()
	smg := &SousManifestGet{
		TargetManifestID: graph.TargetManifestID{
			Source: sous.SourceLocation{
				Repo: project1.Repo,
			},
			Flavor: "chocolate",
		},
		HTTPClient: graph.HTTPClient{cl},

		OutWriter: graph.OutWriter(out),
		LogSet:    logging.NewLogSet("", os.Stderr),
	}

	control.Any(
		"Retrieve",
		testManifest("simple"), restfultest.DummyUpdater(), nil,
	)

	res := smg.Execute([]string{})
	assert.Equal(t, 0, res.ExitCode())

	if assert.Len(t, control.Calls(), 1) {
		assert.Regexp(t, "/manifests", control.Calls()[0].PassedArgs().String(0))
		params := control.Calls()[0].PassedArgs().Get(1).(map[string]string)
		assert.Contains(t, params, "repo")
		assert.Contains(t, params, "flavor")
		assert.Equal(t, params["flavor"], "chocolate")
	}

	assert.Regexp(t, "github", out.String())
}

func TestManifestSet(t *testing.T) {
	cl, control := restfultest.NewHTTPClientSpy()
	mid := sous.ManifestID{
		Source: sous.SourceLocation{
			Repo: project1.Repo,
		},
	}

	mani := testManifest("simple")

	mani.Flavor = "vanilla"
	yml, err := yaml.Marshal(mani)
	require.NoError(t, err)
	in := bytes.NewBuffer(yml)

	sms := &SousManifestSet{
		TargetManifestID: graph.TargetManifestID(mid),

		HTTPClient: graph.HTTPClient{cl},

		InReader: graph.InReader(in),
		LogSet:   logging.NewLogSet("", os.Stderr),
	}

	updater, upctl := restfultest.NewUpdateSpy()
	control.MatchMethod(
		"Retrieve",
		spies.Once(),
		testManifest("simple"), updater, nil,
	)
	control.Any(
		"Retrieve",
		testManifest("simple"), restfultest.DummyUpdater(), nil,
	)
	upctl.Any(
		"Update",
		nil,
	)

	res := sms.Execute([]string{})
	assert.Equal(t, 0, res.ExitCode())

	if assert.Len(t, control.Calls(), 1) {
		args := control.Calls()[0].PassedArgs()
		assert.Regexp(t, "/manifests", args.String(0))
	}
	if assert.Len(t, upctl.Calls(), 1) {
		args := upctl.Calls()[0].PassedArgs()
		assert.Equal(t, args.Get(0).(*sous.Manifest).Flavor, "vanilla")
	}
}

func TestManifestYAML(t *testing.T) {
	uripath := "certainly/i/am/healthy"
	yml, err := yaml.Marshal(testManifest("simple"))
	require.NoError(t, err)
	assert.Regexp(t, "(?i).*checkready.*", string(yml))

	newM := sous.Manifest{}
	err = yaml.Unmarshal(yml, &newM)
	require.NoError(t, err)

	assert.Equal(t, newM.Deployments["ci"].Startup.CheckReadyURIPath, uripath)
}
