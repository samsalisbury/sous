package cli

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

var getIDsTests = []struct {
	Flags       DeployFilterFlags
	SL          sous.SourceLocation
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
		SL:          sous.MustParseSourceLocation("github.com/blah/blah"),
		ExpectedSID: sous.MustParseSourceID("github.com/blah/blah,1.0.0"),
		ExpectedDID: sous.DeployID{Cluster: "blah", Source: sous.SourceLocation{Repo: "github.com/blah/blah"}},
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
	SID                  sous.SourceID
	DID                  sous.DeployID
	ExpectedErr          string
	ExpectedNumManifests int
}{
	{
		State:       sous.NewState(),
		GDM:         CurrentGDM{sous.NewDeployments()},
		ExpectedErr: `cluster "" does not exist`,
	},
	{
		State:       sous.NewState(),
		GDM:         CurrentGDM{sous.NewDeployments()},
		DID:         sous.DeployID{Cluster: "blah"},
		ExpectedErr: `cluster "blah" does not exist`,
	},
	{
		State: &sous.State{
			Defs: sous.Defs{Clusters: sous.Clusters{
				"blah": &sous.Cluster{Name: "blah"},
			}},
		},
		GDM:                  CurrentGDM{sous.NewDeployments()},
		DID:                  sous.DeployID{Cluster: "blah"},
		ExpectedNumManifests: 1,
	},
}

func TestUpdateState(t *testing.T) {
	for _, test := range updateStateTests {
		err := updateState(test.State, test.GDM, test.SID, test.DID)
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
	}
}
