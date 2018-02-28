package server

import (
	"net/http"
	"net/url"
	"strings"
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
		// WantStateWritten true if we expect state to be written.
		WantStateWritten bool
		// WantHeaders is a list of headers we expect in the response.
		WantHeaders http.Header
		// WantQueuedR11n indicates if we expect a R11n to be queued.
		// If true, we assert that a relevant one has been added to the queue,
		// and that the response contains a link to the queued r11n.
		WantQueuedR11n bool
	}{
		{
			Desc: "no matching repo",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				// Return the deployment from the fixture unchanged.
				b := makeBodyFromFixture("github.com/user1/repo1", "cluster1")
				b.DeploymentID.ManifestID.Source.Repo = "nonexistent"
				return b, b.DeploymentID
			},
			WantStatus: 404,
		},
		{
			Desc: "no matching cluster",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture("github.com/user1/repo1", "cluster1")
				b.DeploymentID.Cluster = "nonexistent"
				return b, b.DeploymentID
			},
			WantStatus: 404,
		},
		{
			Desc: "no change necessary",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture("github.com/user1/repo1", "cluster1")
				return b, b.DeploymentID
			},
			WantStatus: 200,
		},
		{
			Desc: "change version",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture("github.com/user1/repo1", "cluster1")
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, b.DeploymentID
			},
			WantStatus:       200,
			WantStateWritten: true,
			WantQueuedR11n:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {

			// Setup

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

			// Shebang

			gotBody, gotStatus := psd.Exchange()

			// Assertions...

			body, ok := gotBody.(*singleDeploymentBody)

			if !ok {
				t.Fatalf("got a %T; want a %T", gotBody, body)
			}

			// TODO SS: Add a diff method to singleDeploymentBody to print
			// specific diffs for ease of reading test output and also because
			// we may want to add metadata that does not participate in equality
			// checks later.
			if gotBody != sent {
				t.Errorf("received != sent:\nreceived:\n%#v\n\nsent:\n%#v",
					gotBody, sent) // Horror blob see todo above.
			}

			if gotStatus != tc.WantStatus {
				t.Errorf("got status %d; want %d", gotStatus, tc.WantStatus)
			}

			gotStateWritten := len(stateWriter.Spy.CallsTo("WriteState")) == 1
			if gotStateWritten != tc.WantStateWritten {
				t.Errorf("got state written: %t; want %t", gotStateWritten, tc.WantStateWritten)
			}

			if !tc.WantQueuedR11n {
				return
			}
			t.Run("queued R11n check", func(t *testing.T) {
				qdaLink := "queuedDeployAction"
				gotLink := body.Meta.Links[qdaLink]
				wantPrefix := "/deploy-queue-item"

				if !strings.HasPrefix(gotLink, wantPrefix) {
					t.Fatalf("got Meta.Links[%q] == %q; want prefix %q",
						qdaLink, gotLink, wantPrefix)
				}

				gotURL, err := url.Parse(gotLink)
				if err != nil {
					t.Fatalf("got Meta.Links[%q] == %q; not a valid URL: %s",
						qdaLink, gotLink, err)
				}

				r11nID := sous.R11nID(gotURL.Query().Get("action"))
				if r11nID == "" {
					t.Fatalf("action query param empty")
				}

				q, ok := queueSet.Queues()[body.DeploymentID]
				if !ok {
					t.Fatalf("no queue for %s", body.DeploymentID)
				}
				if _, ok := q.ByID(r11nID); !ok {
					t.Errorf("returned r11n ID %q not queued", r11nID)
				}
			})
		})
	}

}

type stateWriterSpy struct{ *spies.Spy }

func newStateWriterSpy() stateWriterSpy {
	return stateWriterSpy{
		Spy: spies.NewSpy(),
	}
}

func (sw stateWriterSpy) WriteState(s *sous.State, u sous.User) error {
	return sw.Called(s, u).Error(0)
}
