package cli

import (
	"bytes"
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

func TestManifestGet(t *testing.T) {
	out := &bytes.Buffer{}

	cl, control := restfultest.NewHTTPClientSpy()
	control.Any(
		"Retrieve",
		testManifest(), restfultest.DummyUpdater(), nil,
	)

	smg := &SousManifestGet{
		TargetManifestID: graph.TargetManifestID{
			Source: sous.SourceLocation{
				Repo: project1.Repo,
			},
		},
		HTTPClient: graph.HTTPClient{cl},

		OutWriter: graph.OutWriter(out),
		LogSet:    logging.NewLogSet("", os.Stderr),
	}
	res := smg.Execute([]string{})
	assert.Equal(t, 0, res.ExitCode())

	if assert.Len(t, control.Calls(), 1) {
		assert.Regexp(t, "/manifests", control.Calls()[0].PassedArgs().String(0))
		assert.Contains(t, control.Calls()[0].PassedArgs().Get(1).(map[string]string), "repo")
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

	baseMani := testManifest()
	mani := testManifest()

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
		baseMani, updater, nil,
	)
	control.Any(
		"Retrieve",
		baseMani, restfultest.DummyUpdater(), nil,
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

func testManifest() *sous.Manifest {
	uripath := "certainly/i/am/healthy"
	return &sous.Manifest{
		Source: sous.SourceLocation{Repo: project1.Repo},
		Owners: []string{"sam", "judson"},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			"ci": sous.DeploySpec{
				DeployConfig: sous.DeployConfig{
					Resources: sous.Resources{
						"cpus":   "0.1",
						"memory": "100",
						"ports":  "1",
					},
					Startup: sous.Startup{
						CheckReadyURIPath: uripath,
					},
				},
			},
		},
	}
}

func TestManifestYAML(t *testing.T) {
	uripath := "certainly/i/am/healthy"
	yml, err := yaml.Marshal(testManifest())
	require.NoError(t, err)
	assert.Regexp(t, "(?i).*checkready.*", string(yml))

	newM := sous.Manifest{}
	err = yaml.Unmarshal(yml, &newM)
	require.NoError(t, err)

	assert.Equal(t, newM.Deployments["ci"].Startup.CheckReadyURIPath, uripath)
}
