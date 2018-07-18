package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/nyarly/spies"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestSingleDeploymentResource(t *testing.T) {
	qs, _ := sous.NewQueueSetSpy()
	cl := ComponentLocator{
		QueueSet:     qs,
		StateManager: &sous.DummyStateManager{State: sous.DefaultStateFixture()},
	}
	r := newSingleDeploymentResource(cl)

	rm := routemap(cl)

	rw := httptest.NewRecorder()

	t.Run("Get()", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://sous.example.com/single-deployment", nil)

		ls, _ := logging.NewLogSinkSpy()
		gex := r.Get(rm, ls, rw, req, nil)

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
	})

	t.Run("Put()", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "http://sous.example.com/single-deployment", bytes.NewBufferString("{}"))
		ls, _ := logging.NewLogSinkSpy()
		pex := r.Put(rm, ls, rw, req, nil)
		if pex == nil {
			t.Fatalf("r.Put returned nil")
		}

		psdh, is := pex.(*PUTSingleDeploymentHandler)
		if !is {
			t.Fatalf("r.Put did not return a PUTSingleDeploymentHandler")
		}
		if psdh.responseWriter != rw {
			t.Errorf("PUT handler didn't get the ResponseWriter")
		}
		if psdh.req != req {
			t.Errorf("PUT handler didn't get the Request")
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
	handler      *PUTSingleDeploymentHandler
	stateManager *sous.DummyStateManager
	gdm          *sous.State
	response     interface{}
	status       int
	queueSet     *spies.Spy
}

func (scn *psdhExScenario) exercise() {
	scn.response, scn.status = scn.handler.Exchange()
}

func (scn psdhExScenario) assertStatus(t *testing.T, expected int) {
	t.Helper()
	if scn.status != expected {
		t.Errorf("Expected status %d, got %d", expected, scn.status)
	}
}

func (scn psdhExScenario) assertHeader(t *testing.T, wantKey, wantValue string) {
	t.Helper()

	getHeader, ok := scn.response.(restful.HeaderAdder)
	if !ok {
		t.Errorf("no header")
		return
	}
	h := http.Header{}
	getHeader.AddHeaders(h)

	gotValue := h.Get(wantKey)
	if gotValue != wantValue {
		t.Errorf("got:\n%q=%q\nwant:\n%q=%q", wantKey, gotValue, wantKey, wantValue)
	}
}

func (scn psdhExScenario) assertStringBody(t *testing.T, expected string) {
	t.Helper()
	body, is := scn.response.(string)
	if !is {
		t.Errorf("Expected a simple string response - got %T", scn.response)
		return
	}
	if !strings.Contains(body, expected) {
		t.Errorf("Expected response to contain %q, but not found in %q", expected, body)
	}
}

func (scn psdhExScenario) assertDeploymentWritten(t *testing.T) {
	t.Helper()
	if scn.stateManager.WriteCount != 1 {
		t.Errorf("Expected that a deployment would be written once; written %d times.", scn.stateManager.WriteCount)
	}
}

func (scn psdhExScenario) assertR11nQueued(t *testing.T) {
	t.Helper()
	calls := scn.queueSet.CallsTo("Push")
	if len(calls) == 0 {
		t.Errorf("Expected that a rectification would be queued, but none were.")
	}
}

func (scn psdhExScenario) assertNoR11nQueued(t *testing.T) {
	t.Helper()
	piecalls := scn.queueSet.CallsTo("PushIfEmpty")
	pcalls := scn.queueSet.CallsTo("Push")
	if len(piecalls) != 0 || len(pcalls) != 0 {
		t.Errorf("Expected that no rectification would be queued, but one was.")
	}
}

func TestPUTSingleDeploymentHandler_Exchange(t *testing.T) {
	setup := func(sent *SingleDeploymentBody, did map[string]string) *psdhExScenario {
		qs, qsCtrl := sous.NewQueueSetSpy()
		sm := &sous.DummyStateManager{State: sous.DefaultStateFixture()}
		log, _ := logging.NewLogSinkSpy()
		cl := ComponentLocator{
			StateManager: sm,
			QueueSet:     qs,
			LogSink:      log,
		}
		r := newSingleDeploymentResource(cl)

		rm := routemap(cl)

		values := url.Values{}
		for k, v := range did {
			values.Add(k, v)
		}
		url, err := url.Parse("http://sous.example.com/single-deployment?" + values.Encode())
		if err != nil {
			t.Fatalf("Error parsing URL during setup: %v", err)
		}

		bs, err := json.Marshal(sent)
		if err != nil {
			t.Fatalf("Error marshalling test sent body: %v", err)
		}
		body := bytes.NewBuffer(bs)

		req := httptest.NewRequest("PUT", url.String(), body)
		req.Header.Set("Sous-User-Name", "Test User")
		req.Header.Set("Sous-User-Email", "testuser@example")

		rw := httptest.NewRecorder()

		ls, _ := logging.NewLogSinkSpy()
		psd := r.Put(rm, ls, rw, req, nil).(*PUTSingleDeploymentHandler)

		return &psdhExScenario{
			handler:      psd,
			stateManager: sm,
			queueSet:     qsCtrl,
			gdm:          psd.GDM,
		}
	}

	didQuery := func(repo, offset, cluster, flavor, force string) map[string]string {
		return map[string]string{
			"repo":    repo,
			"offset":  offset,
			"cluster": cluster,
			"flavor":  flavor,
			"force":   force,
		}
	}

	makeBodyAndQuery := func(t *testing.T, force bool) (*SingleDeploymentBody, map[string]string) {
		t.Helper()
		m, ok := sous.DefaultStateFixture().Manifests.Get(
			sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user1/repo1",
					Dir:  "dir1",
				},
				Flavor: "flavor1",
			},
		)
		if !ok {
			t.Fatal("Setup failed to get Manifest.")
		}
		dep, ok := m.Deployments["cluster1"]
		if !ok {
			t.Fatal("Setup failed to get DeploySpec.")
		}
		fstr := strconv.FormatBool(force)
		query := didQuery(m.Source.Repo, m.Source.Dir, "cluster1", m.Flavor, fstr)

		return &SingleDeploymentBody{Deployment: &dep}, query
	}

	t.Run("query parsing error", func(t *testing.T) {
		scenario := setup(nil, map[string]string{})
		scenario.exercise()

		scenario.assertStatus(t, 400)
		scenario.assertStringBody(t, `Cannot decode Deployment ID:`)
	})

	t.Run("body parsing error", func(t *testing.T) {
		scenario := setup(nil, didQuery("github.com/opentable/something", "", "cluster", "", "false"))
		scenario.exercise()

		scenario.assertStatus(t, 400)
		scenario.assertStringBody(t, `Body.Deployment is nil.`)
	})

	t.Run("nonexistent cluster", func(t *testing.T) {
		body, query := makeBodyAndQuery(t, false)
		query["cluster"] = "nonexistent_cluster"
		scenario := setup(body, query)
		scenario.exercise()
		scenario.assertStatus(t, 404)
		scenario.assertStringBody(t, `Cluster "nonexistent_cluster" not defined.`)
	})

	t.Run("no matching deployment", func(t *testing.T) {
		body, query := makeBodyAndQuery(t, false)
		scenario := setup(body, query)
		mid := sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: "github.com/user1/repo1",
				Dir:  "dir1",
			},
			Flavor: "flavor1",
		}
		m, ok := scenario.gdm.Manifests.Get(mid)
		if !ok {
			t.Fatal("Setup failed to get manifest.")
		}
		m.Deployments = sous.DeploySpecs{}
		scenario.exercise()

		scenario.assertStatus(t, 404)
		wantErr := `Manifest "github.com/user1/repo1,dir1~flavor1" has no deployment for cluster "cluster1".`
		scenario.assertStringBody(t, wantErr)
	})

	t.Run("change version", func(t *testing.T) {
		body, query := makeBodyAndQuery(t, false)
		body.Deployment.Version = semv.MustParse("2.0.0")
		scenario := setup(body, query)
		qr := &sous.QueuedR11n{
			ID: "actionid1",
		}
		scenario.queueSet.MatchMethod("Push", spies.AnyArgs, qr, true)
		scenario.exercise()

		scenario.assertStatus(t, 201)
		scenario.assertDeploymentWritten(t)
		scenario.assertR11nQueued(t)
		scenario.assertHeader(t, "Location",
			"sous.example.com/deploy-queue-item?action=actionid1&cluster=cluster1&flavor=flavor1&offset=dir1&repo=github.com%2Fuser1%2Frepo1")
	})

	t.Run("WriteDeployment error", func(t *testing.T) {
		body, query := makeBodyAndQuery(t, false)
		body.Deployment.NumInstances = 7
		scenario := setup(body, query)

		scenario.stateManager.WriteErr = errors.New("an error occurred")
		scenario.exercise()

		scenario.assertDeploymentWritten(t)
		scenario.assertStatus(t, 500)
		scenario.assertStringBody(t, "Failed to write state: an error occurred.")
	})

	t.Run("PushToQueueSet error", func(t *testing.T) {
		body, query := makeBodyAndQuery(t, false)
		body.Deployment.NumInstances = 7
		scenario := setup(body, query)
		scenario.queueSet.MatchMethod("Push", spies.AnyArgs, &sous.QueuedR11n{}, false)
		scenario.exercise()

		scenario.assertDeploymentWritten(t)
		scenario.assertStatus(t, 409)
		scenario.assertStringBody(t, "Queue full, please try again later.")
	})

	t.Run("same_version_force_false", func(t *testing.T) {
		body, query := makeBodyAndQuery(t, false)
		body.Deployment.Version = semv.MustParse("1.0.0")
		scenario := setup(body, query)
		qr := &sous.QueuedR11n{
			ID: "actionid1",
		}
		scenario.queueSet.MatchMethod("Push", spies.AnyArgs, qr, true)
		scenario.exercise()

		//GDM correct, returns 200
		scenario.assertStatus(t, 200)
	})

	t.Run("same_version_force_true", func(t *testing.T) {
		body, query := makeBodyAndQuery(t, true)

		//same version, force true, redeploy
		body.Deployment.Version = semv.MustParse("1.0.0")
		scenario := setup(body, query)
		qr := &sous.QueuedR11n{
			ID: "actionid1",
		}
		scenario.queueSet.MatchMethod("Push", spies.AnyArgs, qr, true)
		scenario.exercise()

		scenario.assertStatus(t, 201)
		scenario.assertDeploymentWritten(t)
		scenario.assertR11nQueued(t)
		scenario.assertHeader(t, "Location",
			"sous.example.com/deploy-queue-item?action=actionid1&cluster=cluster1&flavor=flavor1&offset=dir1&repo=github.com%2Fuser1%2Frepo1")
	})

}

func TestMakeSingularityURL_valid(t *testing.T) {
	baseURL := "http://a.b"
	requestID := "123-4"
	var deploymentID sous.DeploymentID

	url := makeSingularityURL(baseURL, requestID, deploymentID)
	assert.Equal(t, "http://a.b/request/123-4", url, "url's should match")

	deploymentID = sous.DeploymentID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Dir:  "test-dir",
				Repo: "github.com/opentable/test.git",
			},
			Flavor: "test-flavor",
		},
		Cluster: "test-cluster",
	}
	expectedURL := fmt.Sprintf("http://a.b/request/test_git-test_dir-test_flavor-test_cluster-%x", deploymentID.Digest())
	url = makeSingularityURL(baseURL, "", deploymentID)
	assert.Equal(t, expectedURL, url, "url's should match")
}

func TestMakeSingularityURL_invalid(t *testing.T) {
	baseURL := ""
	requestID := "123-4"
	var deploymentID sous.DeploymentID

	url := makeSingularityURL(baseURL, requestID, deploymentID)
	assert.Equal(t, "Unable to determine SingularityRequest URL : baseURL can not be empty : ", url)
	url = makeSingularityURL(baseURL, "", deploymentID)
	assert.Equal(t, "Unable to determine SingularityRequest URL : string \"\" does not represent a repo", url)
	url = makeSingularityURL("", "", deploymentID)
	assert.Equal(t, "Unable to determine SingularityRequest URL : string \"\" does not represent a repo", url)
}
