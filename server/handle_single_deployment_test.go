package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/test"
)

func TestSingleDeploymentResource(t *testing.T) {
	cl := ComponentLocator{
		DeploymentManager: sous.MakeDeploymentManager(sm),
		QueueSet:          qs,
	}
	r := newSingleDeploymentResource(cl)

	rm := routemap(cl)

	rw := httptest.NewRecorder()

	t.Run("Get()", func(t *testing.T) {
		req := httptest.NewRequest("GET", "./single-deployment", nil)

		gex := r.Get(rm, rw, req, nil)

		if gex == nil {
			t.Fatalf("r.Get returned nil")
		}

		gsdh, is := gex.(*GETSingleDeploymentHandler)
		if !is {
			t.Fatalf("r.GET did not return a GETSingleDeploymentHandler")
		}
		if gsdh.responseWriter != rw {
			t.Errorf("GET handler didn't get the ResponseWriter")
		}
		if gsdh.req != req {
			t.Errorf("GET handler didn't get the Request")
		}
		if gsdh.DeploymentManager != cl.DeploymentManager {
			t.Errorf("GET handler didn't get the DeploymentManager")
		}
	})

	t.Run("Put()", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "./single-deployment", bytes.NewBufferString("{}"))
		pex := r.Put(rm, rw, req, nil)
		if pex == nil {
			t.Fatalf("r.Get returned nil")
		}

		psdh, is := pex.(*PUTSingleDeploymentHandler)
		if !is {
			t.Fatalf("r.PUT did not return a PUTSingleDeploymentHandler")
		}
		if psdh.responseWriter != rw {
			t.Errorf("PUT handler didn't get the ResponseWriter")
		}
		if psdh.req != req {
			t.Errorf("PUT handler didn't get the Request")
		}
		if psdh.DeploymentManager != cl.DeploymentManager {
			t.Errorf("PUT handler didn't get the DeploymentManager")
		}
		if psdh.QueueSet != cl.QueueSet {
			t.Errorf("PUT handler didn't get the QueueSet")
		}
		if psdh.routeMap != rm {
			t.Errorf("PUT handler didn't get the route map")
		}
	})

}

type psdhExScenario struct {
	response interface{}
	status   int
}

func TestPUTSingleDeploymentHandler_Exchange(t *testing.T) {
	setup := func(sent *SingleDeploymentBody, did map[string]string) {
		// Setup

		stateWriter := newStateWriterSpy()
		queueSet := sous.NewR11nQueueSet()
		user := sous.User{
			Name:  "Test User",
			Email: "testuser@example.com",
		}

		state := test.DefaultStateFixture()
		stateToDeployments := func(s *sous.State) (sous.Deployments, error) {
			return state.Deployments()
		}

		rm := routemap(ComponentLocator{})

		cl := ComponentLocator{
			DeploymentManager: sous.MakeDeploymentManager(sm),
			QueueSet:          qs,
		}
		r := newSingleDeploymentResource(cl)

		rm := routemap(cl)

		values := url.Values{}
		for k, v := range did {
			values.Add(k, v)
		}
		url := url.Parse("http://sous.example.com/single-deployment?" + values.Encode())

		bytes, err := json.Marshal(sent)
		if err != nil {
			t.Fatalf("Error marshalling test sent body: %v", err)
		}
		body := bytes.NewBuffer(bytes)

		req := httptest.NewRequest("PUT", "./single-deployment", body)

		psd := r.Put(rm, req, rw, nil)

		response, status := psd.Exchange()
		return psdhExScenario{
			response: response,
			status:   status,
		}
	}

	assertStatus := func(t *testing.T, expected int, scenario psdhExScenario) {
		t.Helper()
		if scenario.status != expected {
			t.Errorf("Expected status %d, got %d", expected, scenario.status)
		}
	}

	assertStringBody(t *testing.T, expected string, scenario) {
		t.Helper()
		body, is := scenario.response.(string)
		if !is {
			t.Errorf("Expected a simple string response - got %T", scenario.response)
			return
		}
		if !strings.Contains(body, expected) {
			t.Errorf("Expected response to contain %q, but not found in %q", expected, body)
		}
	}

	didQuery := func(repo, offset, cluster, flavor string) map[string]string {
		return map[string]string{
			"repo": repo,
			"offset": offset,
			"cluster": cluster,
			"flavor": flavor,
		}
	}

	t.Run("body parsing error", func(t *testing.T) {
		scenario := setup(nil, map[string]string{})
		assertStatus(t, 400, scenario)
		assertStringBody(t, `Error parsing body: body parse error.`, scenario)
	})

	t.Run("no matching repo", func(t *testing.T) {
		scenario := setup(&SingleDeploymentBody{Deployment: sous.DeploymentFixture("")}, didQuery("nonexistent", "", "cluster-1", ""))
		assertStatus(t, 404, scenario)
		assertStringBody(t, "No manifest", scenario)
	})

	t.Run( "no matching cluster", func(t *testing.T) {
		BodyAndID: func() (*SingleDeploymentBody, sous.DeploymentID) {
		scenario := setup(&SingleDeploymentBody{Deployment: sous.DeploymentFixture("")}, didQuery("github.com/user1/repo1", "", "nonexistent", ""))
		assertStatus(t, 404, scenario)
		assertStringBody(t, "deployment defined", scenario)
	})
}

/*
	makeBodyFromFixture := func(t *testing.T, repo, cluster string) (*SingleDeploymentBody, sous.DeploymentID) {
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

		return &SingleDeploymentBody{
				ManifestHeader: *m,
				DeploySpec:     d,
			},
			sous.DeploymentID{
				ManifestID: m.ID(),
				Cluster:    cluster,
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
		BodyAndID func() (*SingleDeploymentBody, sous.DeploymentID)
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
		{
			Desc: "no matching cluster",
			BodyAndID: func() (*SingleDeploymentBody, sous.DeploymentID) {
				b, did := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				// Set bogus cluster.
				did.Cluster = "nonexistent"
				return b, did
			},
			WantStatus: 404,
			WantError:  `No "nonexistent" deployment defined for "nonexistent:github.com/user1/repo1,dir1~flavor1".`,
		},
		{
			Desc: "no change necessary",
			BodyAndID: func() (*SingleDeploymentBody, sous.DeploymentID) {
				b, did := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				return b, did
			},
			WantStatus: 200,
		},
		{
			Desc: "change version",
			BodyAndID: func() (*SingleDeploymentBody, sous.DeploymentID) {
				b, did := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, did
			},
			WantStatus:           201,
			WantWriteStateCalled: true,
			WantQueuedR11n:       true,
		},
		{
			Desc: "StateWriter.Write error",
			BodyAndID: func() (*SingleDeploymentBody, sous.DeploymentID) {
				b, did := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				// Make a change to trigger write attempt.
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, did
			},
			OverrideStateWriter:  newStateWriterSpyWithError("an error occured"),
			WantStatus:           500,
			WantWriteStateCalled: true,
			WantError:            "Failed to write state: an error occured.",
		},
		{
			Desc: "State.Deployments error",
			BodyAndID: func() (*SingleDeploymentBody, sous.DeploymentID) {
				b, did := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				// Make a change to trigger write attempt.
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, did
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
			BodyAndID: func() (*SingleDeploymentBody, sous.DeploymentID) {
				b, did := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				// Make a change to trigger write attempt.
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, did
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
			BodyAndID: func() (*SingleDeploymentBody, sous.DeploymentID) {
				b, did := makeBodyFromFixture(t, "github.com/user1/repo1", "cluster1")
				// Make a change to trigger write attempt.
				b.DeploySpec.Version = semv.MustParse("2.0.0")
				return b, did
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

			// Assertions...

			body, ok := gotBody.(*SingleDeploymentBody)

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

				q, ok := queueSet.Queues()[psd.DeploymentID]
				if !ok {
					t.Fatalf("no queue for %s", psd.DeploymentID)
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
*/
