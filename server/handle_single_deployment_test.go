package server

import (
	"net/http"
	"testing"

	"github.com/nyarly/spies"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/test"
	"github.com/samsalisbury/semv"
)

func TestPUTSingleDeploymentHandler_Exchange(t *testing.T) {

	// makeBodyFromFixture returns a body derived from data in the test fixture.
	makeBodyFromFixture := func(repo, cluster string) *singleDeploymentBody {
		t.Helper()
		state := test.DefaultStateFixture()
		m, ok := state.Manifests.Single(func(m *sous.Manifest) bool {
			return m.Source.Repo == repo
		})
		if !ok {
			t.Fatalf("setup failed: no manifest with repo %q in default fixture", repo)
		}
		d, ok := m.Deployments[cluster]
		if !ok {
			t.Fatalf("setup failed: manifest %q has no deployment %q in default fixture", repo, cluster)
		}
		m.Deployments = nil
		return &singleDeploymentBody{
			ManifestHeader: *m,
			DeploymentID: sous.DeploymentID{
				ManifestID: m.ID(),
				Cluster:    cluster,
			},
			DeploySpec: d,
		}
	}

	testCases := []struct {
		// Desc is a short unique description of the test case.
		Desc string
		// BodyAndID is a function that generates a body and an ID.
		// We expect that if response.DeploymentID == id and the server is
		// configured to service requests from the corresponding cluster,
		// the GDM should be updated and a new R11n enqueued.
		//
		// The body is sent as the PUT body of the request.
		// We expect that the same body is returned on success.
		BodyAndID func() (*singleDeploymentBody, sous.DeploymentID)
		// WantStatus is the expected HTTP status for this request.
		WantStatus int
	}{
		{
			Desc: "no matching repo",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				// We return the deployment from the fixture unchanged.
				b := makeBodyFromFixture("github.com/user1/repo1", "cluster1")
				b.DeploymentID.ManifestID.Source.Repo = "nonexistent"
				return b, b.DeploymentID
			},
			WantStatus: 404,
		},
		{
			Desc: "no matching cluster",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				// We return the deployment from the fixture unchanged.
				b := makeBodyFromFixture("github.com/user1/repo1", "cluster1")
				b.DeploymentID.Cluster = "nonexistent"
				return b, b.DeploymentID
			},
			WantStatus: 404,
		},
		{
			Desc: "no change necessary",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				// We return the deployment from the fixture unchanged.
				b := makeBodyFromFixture("github.com/user1/repo1", "cluster1")
				return b, b.DeploymentID
			},
			WantStatus: 200,
		},
		{
			Desc: "change version",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				// We return the deployment from the fixture unchanged.
				b := makeBodyFromFixture("github.com/user1/repo1", "cluster1")
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, b.DeploymentID
			},
			WantStatus: 200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {

			sent, did := tc.BodyAndID()
			header := http.Header{}
			stateWriter := newStateWriterSpy()
			queueSet := sous.NewR11nQueueSet()
			user := sous.User{
				Name:  "Test User",
				Email: "testuser@example.com",
			}

			psd := PUTSingleDeploymentHandler{
				DeploymentID: did,
				Body:         sent,
				GDM:          test.DefaultStateFixture(),
				Header:       header,
				StateWriter:  stateWriter,
				QueueSet:     queueSet,
				User:         user,
			}

			received, gotStatus := psd.Exchange()

			got, ok := received.(*singleDeploymentBody)

			if !ok {
				t.Fatalf("got a %T; want a %T", received, got)
			}

			if gotStatus != tc.WantStatus {
				t.Errorf("got status %d; want %d", gotStatus, tc.WantStatus)
			}

			// TODO SS: Add a diff method to singleDeploymentBody to print
			// specific diffs for ease of reading test output and also because
			// we may want to add metadata that does not participate in equality
			// checks later.
			if received != sent {
				t.Errorf("received != sent:\nreceived:\n%#v\n\nsent:\n%#v",
					received, sent) // Horror blob see todo above.
			}
		})
	}

}

type stateWriterSpy struct {
	*spies.Spy
}

func newStateWriterSpy() stateWriterSpy {
	return stateWriterSpy{spies.NewSpy()}
}

func (sw stateWriterSpy) WriteState(s *sous.State, u sous.User) error {
	return sw.Called(s, u).Error(0)
}
