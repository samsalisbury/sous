package actions

import (
	"testing"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateStateTests = []struct {
	State                *sous.State
	GDM                  sous.Deployments
	DID                  sous.DeploymentID
	ExpectedErr          string
	ExpectedNumManifests int
}{
	{
		State:       sous.NewState(),
		GDM:         sous.NewDeployments(),
		ExpectedErr: "invalid deploy ID (no cluster name)",
	},
	{
		State: sous.NewState(),
		GDM:   sous.NewDeployments(),
		DID: sous.DeploymentID{
			Cluster:    "blah",
			ManifestID: sous.MustParseManifestID("github.com/user/project"),
		},
		ExpectedErr: `cluster "blah" is not described in defs.yaml`,
	},
	{
		State: &sous.State{
			Defs: sous.Defs{Clusters: sous.Clusters{
				"blah": &sous.Cluster{Name: "blah"},
			}},
			Manifests: sous.NewManifests(),
		},
		GDM: sous.NewDeployments(),
		DID: sous.DeploymentID{
			Cluster:    "blah",
			ManifestID: sous.MustParseManifestID("github.com/user/project"),
		},
		ExpectedNumManifests: 1,
	},
}

func TestUpdateState(t *testing.T) {
	for _, test := range updateStateTests {
		sid := sous.MustNewSourceID(test.DID.ManifestID.Source.Repo, test.DID.ManifestID.Source.Dir, "1.0.0")

		ls, _ := logging.NewLogSinkSpy()
		err := updateState(test.State, test.GDM, sid, test.DID, ls)
		if err != nil {
			if test.ExpectedErr == "" {
				t.Error(err)
				continue
			}
			errStr := err.Error()
			if errStr != test.ExpectedErr {
				t.Errorf("got error %q; want %q", errStr, test.ExpectedErr)
			}
			continue
		}
		if test.ExpectedErr != "" {
			t.Errorf("got nil; want error %q", test.ExpectedErr)
		}
		actualNumManifests := test.State.Manifests.Len()
		if actualNumManifests != test.ExpectedNumManifests {
			t.Errorf("got %d manifests; want %d", actualNumManifests, test.ExpectedNumManifests)
		}
		if (test.DID != sous.DeploymentID{}) {
			m, ok := test.State.Manifests.Get(test.DID.ManifestID)
			if !ok {
				t.Errorf("manifest %q not found", sid.Location)
			}
			_, ok = m.Deployments[test.DID.Cluster]
			if !ok {
				t.Errorf("missing deployment %q", test.DID)
			}
		}
	}
}

func TestUpdateRetryLoop(t *testing.T) {
	/*
		Source SourceLocation `validate:"nonzero"`
		Flavor string `yaml:",omitempty"`
		Owners []string
		Kind ManifestKind `validate:"nonzero"`
		Deployments DeploySpecs `validate:"keys=nonempty,values=nonzero"`
	*/
	depID := sous.DeploymentID{Cluster: "test-cluster", ManifestID: sous.MustParseManifestID("github.com/user/project")}
	sourceID := sous.MustNewSourceID("github.com/user/project", "", "1.2.3")
	mani := &sous.Manifest{
		Source: sourceID.Location,
		Kind:   sous.ManifestKindService,

		Deployments: sous.DeploySpecs{
			"test-cluster": {
				Version: semv.MustParse("0.0.0"),
				DeployConfig: sous.DeployConfig{
					Resources: sous.Resources{
						"cpus":   "1",
						"memory": "100",
						"ports":  "1",
					},
					Startup: sous.Startup{SkipCheck: true},
				},
			},
		},
	}
	t.Log(mani.ID())
	user := sous.User{Name: "Judson the Unlucky", Email: "unlucky@opentable.com"}

	cl, control, err := server.TestingInMemoryClient()
	require.NoError(t, err)

	ls := logging.SilentLogSet()
	tid := sous.TraceID("test-trace")

	hsm := sous.NewHTTPStateManager(cl, tid, ls)

	control.State.Manifests.Add(mani)

	deps, err := updateRetryLoop(ls, hsm, sourceID, depID, user)

	assert.NoError(t, err)
	assert.Equal(t, 1, deps.Len())
	dep, present := deps.Get(depID)
	require.True(t, present)
	assert.Equal(t, "1.2.3", dep.SourceID.Version.String())
	//assert.True(t, dsm.ReadCount > 0, "No requests made against state manager")
}

//XXX should actually drive interesting behavior
func TestSousUpdate_Execute(t *testing.T) {
	cl, control, err := server.TestingInMemoryClient()
	require.NoError(t, err)
	ls, _ := logging.NewLogSinkSpy()
	hsm := sous.NewHTTPStateManager(cl, sous.TraceID("test-trace"), ls)

	manifest := sous.Manifest{
		Source: sous.SourceLocation{
			Repo: "github.com/example/project",
			Dir:  "",
		},
		Flavor: "",
		Kind:   sous.ManifestKindService,
		Deployments: map[string]sous.DeploySpec{
			"test-cluster": {
				DeployConfig: sous.DeployConfig{
					Resources: sous.Resources{
						"cpus":   "1",
						"memory": "100",
						"ports":  "1",
					},
					NumInstances: 1,
					Startup: sous.Startup{
						SkipCheck: true,
					},
				},
				Version: semv.MustParse("0.0.1"),
			},
		},
	}

	control.State.Defs = sous.Defs{
		DockerRepo: "",
		Clusters: map[string]*sous.Cluster{
			"test-cluster": {
				Name:    "test-cluster",
				Kind:    "singularity",
				BaseURL: "test-cluster.example.com",
				Startup: sous.Startup{
					SkipCheck: true,
				},
				AllowedAdvisories: nil,
			},
		},
	}

	control.State.Manifests.Add(&manifest)

	dff := config.DeployFilterFlags{
		DeploymentIDFlags: config.DeploymentIDFlags{
			Cluster: "test-cluster",
			ManifestIDFlags: config.ManifestIDFlags{
				Flavor: manifest.Flavor,
				SourceLocationFlags: config.SourceLocationFlags{
					Repo:   manifest.Source.Repo,
					Offset: manifest.Source.Dir,
				}}},
		SourceVersionFlags: config.SourceVersionFlags{
			Tag: manifest.Deployments["test-cluster"].Version.String(),
		},
	}

	shc := sous.SourceHostChooser{}

	filter, err := dff.BuildFilter(shc.ParseSourceLocation)
	require.NoError(t, err)

	gdm, err := control.State.Deployments()
	require.NoError(t, err)

	su := Update{
		//StateManager:  &graph.StateManager{dsm},
		Manifest:         &manifest,
		GDM:              gdm,
		HTTPStateManager: hsm,
		ResolveFilter:    filter,
		User:             sous.User{},
		Log:              control.Log,
	}
	assert.NoError(t, su.Do())
}
