package cli

import (
	"testing"

	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
)

type getIDsTestCase struct {
	Flags       sous.ResolveFilter
	SL          sous.ManifestID
	ExpectedSID sous.SourceID
	ExpectedDID sous.DeployID
	ExpectedErr string
}

func TestGetIDs(t *testing.T) {
	test := func(test getIDsTestCase) {
		sid, did, err := getIDs(&test.Flags, test.SL)
		if err != nil {
			if test.ExpectedErr == "" {
				t.Error(err)
			} else {
				actualErr := err.Error()
				if actualErr != test.ExpectedErr {
					t.Errorf("got error %q; want %q", actualErr, test.ExpectedErr)
				}
			}
		}
		if err == nil && test.ExpectedErr != "" {
			t.Errorf("got nil; want error %q", test.ExpectedErr)
		}
		if sid != test.ExpectedSID {
			t.Errorf("got SourceID %q; want %q", sid, test.ExpectedSID)
		}
		if did != test.ExpectedDID {
			t.Errorf("got DeployID %q; want %q", did, test.ExpectedDID)
		}
	}

	test(getIDsTestCase{
		ExpectedErr: "update: You must select a cluster using the -cluster flag.",
	})
	test(getIDsTestCase{
		Flags:       sous.ResolveFilter{Cluster: "blah"},
		ExpectedErr: "update: You must provide the -tag flag.",
	})
	test(getIDsTestCase{
		Flags:       sous.ResolveFilter{Cluster: "blah", Tag: "nope"},
		ExpectedErr: `update: Version "nope" not valid: unexpected character 'n' at position 0`,
	})
	test(getIDsTestCase{
		Flags:       sous.ResolveFilter{Cluster: "blah", Tag: "nope"},
		ExpectedErr: `update: Version "nope" not valid: unexpected character 'n' at position 0`,
	})
	test(getIDsTestCase{
		Flags:       sous.ResolveFilter{Cluster: "blah", Tag: "1.0.0"},
		SL:          sous.MustParseManifestID("github.com/blah/blah"),
		ExpectedSID: sous.MustParseSourceID("github.com/blah/blah,1.0.0"),
		ExpectedDID: sous.DeployID{Cluster: "blah",
			ManifestID: sous.ManifestID{Source: sous.SourceLocation{Repo: "github.com/blah/blah"}}},
	})
}

var updateStateTests = []struct {
	State                *sous.State
	GDM                  graph.CurrentGDM
	DID                  sous.DeployID
	ExpectedErr          string
	ExpectedNumManifests int
}{
	{
		State:       sous.NewState(),
		GDM:         graph.CurrentGDM{Deployments: sous.NewDeployments()},
		ExpectedErr: "invalid deploy ID (no cluster name)",
	},
	{
		State: sous.NewState(),
		GDM:   graph.CurrentGDM{Deployments: sous.NewDeployments()},
		DID: sous.DeployID{
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
		},
		GDM: graph.CurrentGDM{Deployments: sous.NewDeployments()},
		DID: sous.DeployID{
			Cluster:    "blah",
			ManifestID: sous.MustParseManifestID("github.com/user/project"),
		},
		ExpectedNumManifests: 1,
	},
}

func TestUpdateState(t *testing.T) {
	for _, test := range updateStateTests {
		sid := sous.MustNewSourceID(test.DID.ManifestID.Source.Repo, test.DID.ManifestID.Source.Dir, "1.0.0")
		err := updateState(test.State, test.GDM, sid, test.DID)
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
		if (test.DID != sous.DeployID{}) {
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

type DummyStateManager struct{}

func (dsm *DummyStateManager) WriteState(s *sous.State) error  { return nil }
func (dsm *DummyStateManager) ReadState() (*sous.State, error) { return nil, nil }

//XXX should actually drive interesting behavior
func TestSousUpdate_Execute(t *testing.T) {
	dsm := &DummyStateManager{}
	su := SousUpdate{
		StateReader:   graph.StateReader{StateReader: dsm},
		StateWriter:   graph.StateWriter{StateWriter: dsm},
		GDM:           graph.CurrentGDM{Deployments: sous.MakeDeployments(0)},
		Manifest:      graph.TargetManifest{Manifest: &sous.Manifest{}},
		ResolveFilter: &graph.RefinedResolveFilter{},
	}
	su.Execute(nil)
}
