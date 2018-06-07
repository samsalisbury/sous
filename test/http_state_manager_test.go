package test

import (
	"bytes"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
)

func buildManifest(cluster, repo, version string) *sous.Manifest {
	return &sous.Manifest{
		Owners: []string{"tom", "dick", "harry"},
		Source: sous.SourceLocation{Repo: repo},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			cluster: sous.DeploySpec{
				Version: semv.MustParse(version),
				DeployConfig: sous.DeployConfig{
					Startup: sous.Startup{
						CheckReadyProtocol: "HTTPS",
					},
					Resources: sous.Resources{
						"cpus":   "1",
						"memory": "256",
						"ports":  "1",
					},
				},
			},
		},
	}
}

func TestWriteState(t *testing.T) {
	steadyManifest := buildManifest("test-cluster", "github.com/opentable/steady", "1.2.3")
	diesManifest := buildManifest("test-cluster", "github.com/opentable/dies", "133.56.987431")
	changesManifest := buildManifest("test-cluster", "github.com/opentable/changes", "0.17.19")
	newManifest := buildManifest("test-cluster", "github.com/opentable/new", "0.0.1")

	state := &sous.State{}
	state.SetEtag("qwertybeatsdvorak")
	state.Defs.Clusters = make(sous.Clusters)
	state.Defs.Clusters["test-cluster"] = &sous.Cluster{Name: "test-cluster"}

	// Current issue: "incomplete" manifests never complete to get updates
	// There aren't any deploy specs for extra, which mimics this bug
	state.Defs.Clusters["extra-cluster"] = &sous.Cluster{Name: "cluster-cluster"}

	state.Manifests = sous.NewManifests()
	state.Manifests.Add(steadyManifest)
	state.Manifests.Add(diesManifest)
	state.Manifests.Add(changesManifest)

	sm := sous.DummyStateManager{State: state}

	smm, err := sm.ReadState()
	if err != nil {
		t.Fatal("State manager double is broken", err)
	}
	if smm.Manifests.Len() <= 0 {
		t.Fatal("State manager double is empty")
	}

	db := sous.SetupDB(t)
	defer sous.ReleaseDB(t)

	di := graph.BuildBaseGraph(semv.Version{}, &bytes.Buffer{}, os.Stderr, os.Stderr)
	graph.AddNetwork(di)

	logger, _ := logging.NewLogSinkSpy()
	di.Add(
		func() *config.DeployFilterFlags { return &config.DeployFilterFlags{} },
		func() graph.DryrunOption { return graph.DryrunBoth },

		func() graph.StateReader { return graph.StateReader{StateReader: &sm} },
		func() graph.StateWriter { return graph.StateWriter{StateWriter: &sm} },
		func() *graph.ServerClusterManager {
			return &graph.ServerClusterManager{ClusterManager: sous.MakeClusterManager(&sm, logger)}
		},

		func() *graph.ServerStateManager { return &graph.ServerStateManager{StateManager: &sm} },
		func() *graph.ConfigLoader { return graph.NewTestConfigLoader("") },
		graph.MaybeDatabase{Db: db, Err: nil},
	)

	serverScoop := struct{ Handler graph.ServerHandler }{}
	di.Add(&config.Verbosity{})
	di.MustInject(&serverScoop)
	if serverScoop.Handler.Handler == nil {
		t.Fatalf("Didn't inject http.Handler!")
	}
	testServer := httptest.NewServer(serverScoop.Handler.Handler)
	defer testServer.Close()

	cl, err := restful.NewClient(testServer.URL, logger, map[string]string{"X-Gatelatch": "please"})
	if err != nil {
		t.Fatal(err)
	}
	hsm := sous.NewHTTPStateManager(cl, sous.TraceID("test-trace"), logger)

	originalState, err := hsm.ReadState()
	if err != nil {
		t.Fatal(err)
	}

	for id, m := range originalState.Manifests.Snapshot() {
		t.Logf("hsm INITIAL state: Manifest %q; Kind = %q\n  %#v\n", id, m.Kind, m)
	}

	t.Logf(spew.Sprintf("original state: %#++v", originalState))
	if originalState.Manifests.Len() != state.Manifests.Len() {
		t.Errorf("Local state has %d manifests to remote's %d", originalState.Manifests.Len(), state.Manifests.Len())
	}

	originalState.Manifests.Remove(diesManifest.ID())
	originalState.Manifests.Add(newManifest)
	ch, there := originalState.Manifests.Get(changesManifest.ID())
	if !there {
		t.Fatalf("Changed manifest %q not in local manifests!", changesManifest.ID())
	}
	changedDeployment := ch.Deployments["test-cluster"]
	changedDeployment.Version = semv.MustParse("0.18.0")
	ch.Deployments["test-cluster"] = changedDeployment
	originalState.Manifests.Set(ch.ID(), ch)

	t.Logf("state after update: %#v", originalState)

	var testUser = sous.User{Name: "Test User"}
	if err := hsm.WriteState(originalState, testUser); err != nil {
		t.Fatalf("Failed to write state: %+v", err)
	}

	state, err = hsm.ReadState()
	if err != nil {
		t.Fatal(err)
	}

	for id, m := range state.Manifests.Snapshot() {
		t.Logf("hsm UPDATED state: Manifest %q; Kind = %q\n  %#v\n", id, m.Kind, m)
	}

	if originalState.Manifests.Len() != state.Manifests.Len() {
		t.Errorf("After write, local state has %d manifests to remote's %d", originalState.Manifests.Len(), state.Manifests.Len())
	}

	d, there := state.Manifests.Get(diesManifest.ID())
	if there {
		t.Errorf("Removed manifest still in server's state: %#v", d)
	}
	_, there = state.Manifests.Get(steadyManifest.ID())
	if !there {
		t.Errorf("Untouched manifest not in server's state")
	}
	_, there = state.Manifests.Get(newManifest.ID())
	if !there {
		t.Errorf("Added manifest not in server's state")
	}
	c, there := state.Manifests.Get(changesManifest.ID())
	if !there {
		t.Errorf("Changed manifest missing from server's state")
	}
	expectedVersion := "0.18.0"
	actualVersion := c.Deployments["test-cluster"].Version.String()
	if actualVersion != expectedVersion {
		t.Errorf("Server's version of changed state was %q; want %q", actualVersion, expectedVersion)
	}
}
