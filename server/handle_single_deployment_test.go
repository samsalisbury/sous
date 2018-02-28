package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/nyarly/spies"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/test"
	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

func TestSingleDeploymentResource_Put(t *testing.T) {

	testCases := []struct {
		Desc                string
		URL                 string
		Body                func(t *testing.T) []byte
		WantBodyErr         string
		Header              func(t *testing.T) http.Header
		WantDeploymentID    sous.DeploymentID
		WantDeploymentIDErr string
		WantUser            sous.User
	}{
		{
			Desc: "valid body",
			URL:  "/single-deployment?repo=github.com/user1/repo1&cluster=cluster1",
			Body: func(t *testing.T) []byte {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				j, err := json.Marshal(b)
				if err != nil {
					t.Fatalf("setup failed: %s", err)
				}
				return j
			},
			WantDeploymentID: sous.DeploymentID{
				ManifestID: sous.ManifestID{
					Source: sous.SourceLocation{
						Repo: "github.com/user1/repo1",
					},
				},
				Cluster: "cluster1",
			},
			Header: func(t *testing.T) http.Header {
				h := http.Header{}
				h.Add("Sous-User-Name", "test-user")
				h.Add("Sous-User-Email", "test-user@example.com")
				return h
			},
			WantUser: sous.User{Name: "test-user", Email: "test-user@example.com"},
		},
		{
			Desc: "valid body despite nonexistent cluster",
			URL:  "/single-deployment?repo=github.com/user1/repo1&cluster=blah",
			Body: func(t *testing.T) []byte {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				j, err := json.Marshal(b)
				if err != nil {
					t.Fatalf("setup failed: %s", err)
				}
				return j
			},
			WantDeploymentID: sous.DeploymentID{
				ManifestID: sous.ManifestID{
					Source: sous.SourceLocation{
						Repo: "github.com/user1/repo1",
					},
				},
				Cluster: "blah",
			},
			Header: func(t *testing.T) http.Header {
				h := http.Header{}
				h.Add("Sous-User-Name", "test-user")
				h.Add("Sous-User-Email", "test-user@example.com")
				return h
			},
			WantUser: sous.User{Name: "test-user", Email: "test-user@example.com"},
		},
		{
			Desc: "body is invalid json",
			URL:  "/single-deployment?repo=github.com/user1/repo1&cluster=blah",
			Body: func(t *testing.T) []byte {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				j, err := json.Marshal(b)
				if err != nil {
					t.Fatalf("setup failed: %s", err)
				}
				j[0] = '?' // Make json invalid.
				return j
			},
			WantBodyErr: "invalid character '?' looking for beginning of value",
			WantDeploymentID: sous.DeploymentID{
				ManifestID: sous.ManifestID{
					Source: sous.SourceLocation{
						Repo: "github.com/user1/repo1",
					},
				},
				Cluster: "blah",
			},
			Header: func(t *testing.T) http.Header {
				h := http.Header{}
				h.Add("Sous-User-Name", "test-user")
				h.Add("Sous-User-Email", "test-user@example.com")
				return h
			},
			WantUser: sous.User{Name: "test-user", Email: "test-user@example.com"},
		},
		{
			Desc: "missing cluster",
			URL:  "/single-deployment?repo=github.com/user1/repo1",
			Body: func(t *testing.T) []byte {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				j, err := json.Marshal(b)
				if err != nil {
					t.Fatalf("setup failed: %s", err)
				}
				return j
			},
			WantDeploymentIDErr: "No cluster given",
			Header: func(t *testing.T) http.Header {
				h := http.Header{}
				h.Add("Sous-User-Name", "test-user")
				h.Add("Sous-User-Email", "test-user@example.com")
				return h
			},
			WantUser: sous.User{Name: "test-user", Email: "test-user@example.com"},
		},
		{
			Desc: "missing repo",
			URL:  "/single-deployment?cluster=cluster1",
			Body: func(t *testing.T) []byte {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				j, err := json.Marshal(b)
				if err != nil {
					t.Fatalf("setup failed: %s", err)
				}
				return j
			},
			WantDeploymentIDErr: "No repo given",
			Header: func(t *testing.T) http.Header {
				h := http.Header{}
				h.Add("Sous-User-Name", "test-user")
				h.Add("Sous-User-Email", "test-user@example.com")
				return h
			},
			WantUser: sous.User{Name: "test-user", Email: "test-user@example.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {

			// Setup.
			r := SingleDeploymentResource{}

			u, err := url.Parse(tc.URL)
			if err != nil {
				t.Fatalf("setup failed: %s", err)
			}
			bodyReadCloser := ioutil.NopCloser(bytes.NewBuffer(tc.Body(t)))

			req := &http.Request{
				URL:    u,
				Body:   bodyReadCloser,
				Header: tc.Header(t),
			}

			// Shebang.

			got := r.Put(nil, req, nil).(*PUTSingleDeploymentHandler)

			// Assertions.

			if got.DeploymentID != tc.WantDeploymentID {
				t.Errorf("got deployment ID %q; want %q", got.DeploymentID, tc.WantDeploymentID)
			}

			if got.User != tc.WantUser {
				t.Errorf("got user %# v; want %# v", got.User, tc.WantUser)
			}

			if tc.WantBodyErr != "" {
				gotBodyErr := fmt.Sprint(got.BodyErr)
				if gotBodyErr != tc.WantBodyErr {
					t.Errorf("got body error %q; want %q", gotBodyErr, tc.WantBodyErr)
				}
			} else if got.BodyErr != nil {
				t.Errorf("got body error %q; want nil", got.BodyErr)
			}
			if tc.WantDeploymentIDErr != "" {
				gotDeploymentIDErr := fmt.Sprint(got.DeploymentIDErr)
				if gotDeploymentIDErr != tc.WantDeploymentIDErr {
					t.Errorf("got deployment ID error %q; want %q", gotDeploymentIDErr, tc.WantDeploymentIDErr)
				}
			} else if got.DeploymentIDErr != nil {
				t.Errorf("got deployment ID error %q; want nil", got.DeploymentIDErr)
			}
		})

	}
}

// makeBodyFromFixture returns a body derived from data in the test fixture.
var makeBodyFromFixture = func(t *testing.T, repo, cluster string) *singleDeploymentBody {
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

func TestPUTSingleDeploymentHandler_Exchange(t *testing.T) {

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
		// BodyErrIn is an error parsing a body.
		BodyErrIn error
		// DeploymentIDErr is an error getting valid deployment ID.
		DeploymentIDErr error
		// OverrideGDMToDeployments allows testing for errors.
		OverrideGDMToDeployments func(*sous.State) (sous.Deployments, error)
		// OverrideStateWriter allows using a StateWriter that errors.
		OverrideStateWriter stateWriterSpy
		// OverridePushToQueueSet
		OverridePushToQueueSet func(*sous.Rectification) (*sous.QueuedR11n, bool)
		// WantStatus is the expected HTTP status for this request.
		WantStatus int
		// WantWriteStateCalled true if we expect state to be written.
		WantWriteStateCalled bool
		// WantHeaders is a list of headers we expect in the response.
		WantHeaders http.Header
		// WantQueuedR11n indicates if we expect a R11n to be queued.
		// If true, we assert that a relevant one has been added to the queue,
		// and that the response contains a link to the queued r11n.
		WantQueuedR11n bool
		// WantError is the error message we want to see in meta.
		WantError string
	}{
		{
			Desc: "body parsing error",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				return nil, sous.DeploymentID{}
			},
			BodyErrIn:  errors.New("body parse error"),
			WantStatus: 400,
			WantError:  `Error parsing body: body parse error.`,
		},
		{
			Desc: "no matching repo",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				// Return the deployment from the fixture unchanged.
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				b.DeploymentID.ManifestID.Source.Repo = "nonexistent"
				return b, b.DeploymentID
			},
			WantStatus: 404,
			WantError:  `No manifest with ID "nonexistent,dir1~flavor1".`,
		},
		{
			Desc: "no matching cluster",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				b.DeploymentID.Cluster = "nonexistent"
				return b, b.DeploymentID
			},
			WantStatus: 404,
			WantError:  `No "nonexistent" deployment defined for "nonexistent:github.com/user1/repo1,dir1~flavor1".`,
		},
		{
			Desc: "body deploy ID not match query",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				did := b.DeploymentID
				// cluster2 exists but is not defined in the body.
				did.Cluster = "cluster2"
				return b, did
			},
			WantStatus: 400,
			WantError:  `Body contains deployment "cluster1:github.com/user1/repo1,dir1~flavor1", URL query is for deployment "cluster2:github.com/user1/repo1,dir1~flavor1".`,
		},
		{
			Desc: "no change necessary",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				return b, b.DeploymentID
			},
			WantStatus: 200,
		},
		{
			Desc: "change version",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, b.DeploymentID
			},
			WantStatus:           200,
			WantWriteStateCalled: true,
			WantQueuedR11n:       true,
		},
		{
			Desc: "StateWriter.Write error",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				// Make a change to trigger write attempt.
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, b.DeploymentID
			},
			OverrideStateWriter:  newStateWriterSpyWithError("an error occured"),
			WantStatus:           500,
			WantWriteStateCalled: true,
			WantError:            "Failed to write state: an error occured.",
		},
		{
			Desc: "State.Deployments error",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				// Make a change to trigger write attempt.
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, b.DeploymentID
			},
			OverrideGDMToDeployments: func(*sous.State) (sous.Deployments, error) {
				return sous.Deployments{}, errors.New("an error occured")
			},
			WantStatus:           500,
			WantWriteStateCalled: true,
			WantError:            "Unable to expand GDM: an error occured.",
		},
		{
			Desc: "State.Deployments returns no deployments",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				// Make a change to trigger write attempt.
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, b.DeploymentID
			},
			OverrideGDMToDeployments: func(*sous.State) (sous.Deployments, error) {
				return sous.NewDeployments(), nil
			},
			WantStatus:           500,
			WantWriteStateCalled: true,
			WantError:            "Deployment failed to round-trip to GDM.",
		},
		{
			Desc: "PushToQueueSet error",
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				b := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				// Make a change to trigger write attempt.
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, b.DeploymentID
			},
			OverridePushToQueueSet: func(*sous.Rectification) (*sous.QueuedR11n, bool) {
				return nil, false
			},
			WantStatus:           409,
			WantWriteStateCalled: true,
			WantError:            "Queue full, please try again later.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {

			// Setup

			sent, did := tc.BodyAndID()
			header := http.Header{}
			stateWriter := newStateWriterSpy()
			if tc.OverrideStateWriter != (stateWriterSpy{}) {
				stateWriter = tc.OverrideStateWriter
			}
			queueSet := sous.NewR11nQueueSet()
			user := sous.User{
				Name:  "Test User",
				Email: "testuser@example.com",
			}

			state := test.DefaultStateFixture()
			stateToDeployments := func(s *sous.State) (sous.Deployments, error) {
				return state.Deployments()
			}
			if tc.OverrideGDMToDeployments != nil {
				stateToDeployments = tc.OverrideGDMToDeployments
			}

			pushToQueueSet := queueSet.Push
			if tc.OverridePushToQueueSet != nil {
				pushToQueueSet = tc.OverridePushToQueueSet
			}

			psd := PUTSingleDeploymentHandler{
				DeploymentID:     did,
				Body:             sent,
				BodyErr:          tc.BodyErrIn,
				GDM:              state,
				GDMToDeployments: stateToDeployments,
				Header:           header,
				StateWriter:      stateWriter,
				PushToQueueSet:   pushToQueueSet,
				User:             user,
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
			//if *body != *sent {
			//	t.Errorf("received != sent:\nreceived:\n%#v\n\nsent:\n%#v",
			//		gotBody, sent) // Horror blob see todo above.
			//}

			if gotStatus != tc.WantStatus {
				t.Errorf("got status %d; want %d", gotStatus, tc.WantStatus)
			}
			if body.Meta.StatusCode != gotStatus {
				t.Errorf("got Meta.StatusCode = %d; != actual status code %d",
					body.Meta.StatusCode, gotStatus)
			}

			gotStateWritten := len(stateWriter.Spy.CallsTo("WriteState")) == 1
			if gotStateWritten != tc.WantWriteStateCalled {
				t.Errorf("got state written: %t; want %t", gotStateWritten, tc.WantWriteStateCalled)
			}

			if body.Meta.Error != tc.WantError {
				t.Errorf("got Meta.Error = %q; want %q", body.Meta.Error, tc.WantError)
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

type stateWriterSpy struct {
	Error error
	*spies.Spy
}

func newStateWriterSpy() stateWriterSpy {
	return stateWriterSpy{
		Spy: spies.NewSpy(),
	}
}

func newStateWriterSpyWithError(err string) stateWriterSpy {
	s := newStateWriterSpy()
	s.Error = errors.New(err)
	return s
}

func (sw stateWriterSpy) WriteState(s *sous.State, u sous.User) error {
	sw.Called(s, u)
	return sw.Error
}
