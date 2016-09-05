package cli

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

var getIDsTests = []struct {
	Flags       DeployFilterFlags
	SL          sous.ManifestID
	ExpectedSID sous.SourceID
	ExpectedDID sous.DeployID
	ExpectedErr string
}{
	{
		ExpectedErr: "You must select a cluster using the -cluster flag.",
	},
	{
		Flags:       DeployFilterFlags{Cluster: "blah"},
		ExpectedErr: "You must provide the -tag flag.",
	},
	{
		Flags:       DeployFilterFlags{Cluster: "blah", Tag: "nope"},
		ExpectedErr: `Version "nope" not valid: unexpected character 'n' at position 0`,
	},
	{
		Flags:       DeployFilterFlags{Cluster: "blah", Tag: "nope"},
		ExpectedErr: `Version "nope" not valid: unexpected character 'n' at position 0`,
	},
	{
		Flags:       DeployFilterFlags{Cluster: "blah", Tag: "1.0.0"},
		SL:          sous.MustParseManifestID("github.com/blah/blah"),
		ExpectedSID: sous.MustParseSourceID("github.com/blah/blah,1.0.0"),
		ExpectedDID: sous.DeployID{Cluster: "blah",
			ManifestID: sous.ManifestID{Source: sous.SourceLocation{Repo: "github.com/blah/blah"}}},
	},
}

func TestGetIDs(t *testing.T) {
	for _, test := range getIDsTests {
		sid, did, err := getIDs(test.Flags, test.SL)
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
}

var updateStateTests = []struct {
	State                *sous.State
	GDM                  CurrentGDM
	DID                  sous.DeployID
	ExpectedErr          string
	ExpectedNumManifests int
}{
	{
		State:       sous.NewState(),
		GDM:         CurrentGDM{sous.NewDeployments()},
		ExpectedErr: "invalid deploy ID (no cluster name)",
	},
	{
		State: sous.NewState(),
		GDM:   CurrentGDM{sous.NewDeployments()},
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
		GDM: CurrentGDM{sous.NewDeployments()},
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

func TestSousUpdate_Execute(t *testing.T) {
	dsm := &DummyStateManager{}
	su := SousUpdate{
		StateReader: LocalStateReader{dsm},
		StateWriter: LocalStateWriter{dsm},
		GDM:         CurrentGDM{sous.MakeDeployments(0)},
		Manifest:    TargetManifest{&sous.Manifest{}},
	}
	su.Execute(nil)
}
