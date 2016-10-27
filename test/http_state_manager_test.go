package test

import (
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
)

func buildManifest(cluster, repo, version string) *sous.Manifest {
	return &sous.Manifest{
		Source: sous.SourceLocation{Repo: repo},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			cluster: sous.DeploySpec{Version: semv.MustParse(version)},
		},
	}
}

func TestWriteState(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	steadyManifest := buildManifest("test-cluster", "github.com/opentable/steady", "1.2.3")
	diesManifest := buildManifest("test-cluster", "github.com/opentable/dies", "133.56.987431")
	changesManifest := buildManifest("test-cluster", "github.com/opentable/changes", "0.17.19")
	newManifest := buildManifest("test-cluster", "github.com/opentable/new", "0.0.1")

	state := &sous.State{}
	state.Defs.Clusters = make(sous.Clusters)
	state.Defs.Clusters["test-cluster"] = &sous.Cluster{Name: "test-cluster"}

	state.Manifests = sous.NewManifests()
	state.Manifests.Add(steadyManifest)
	state.Manifests.Add(diesManifest)
	state.Manifests.Add(changesManifest)

	for id, m := range state.Manifests.Snapshot() {
		t.Logf("base state: Manifest %q; Kind = %q", id, m.Kind)
	}

	sm := sous.DummyStateManager{State: state}
	smm, err := sm.ReadState()
	if err != nil {
		t.Fatal("State manager double is broken", err)
	}
	if smm.Manifests.Len() <= 0 {
		t.Fatal("State manager double is empty")
	}

	for id, m := range smm.Manifests.Snapshot() {
		t.Logf("dummy state: Manifest %q; Kind = %q", id, m.Kind)
	}

	gf := func() server.Injector {
		di := psyringe.New()
		di.Add(sous.NewLogSet(os.Stderr, os.Stderr, ioutil.Discard))
		graph.AddInternals(di)
		di.Add(
			func() graph.LocalStateReader { return graph.LocalStateReader{StateReader: sm} },
			func() graph.LocalStateWriter { return graph.LocalStateWriter{StateWriter: sm} },
		)
		di.Add(&config.Verbosity{})
		return di
	}

	testServer := httptest.NewServer(server.SousRouteMap.BuildRouter(gf))
	defer testServer.Close()

	hsm, err := sous.NewHTTPStateManager(testServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	originalState, err := hsm.ReadState()
	if err != nil {
		t.Fatal(err)
	}

	for id, m := range originalState.Manifests.Snapshot() {
		t.Logf("hsm read state: Manifest %q; Kind = %q", id, m.Kind)
	}

	log.Printf("original state: %#v", originalState)
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

	log.Printf("state after update: %#v", originalState)

	if err := hsm.WriteState(originalState); err != nil {
		t.Fatalf("Failed to write state: %+v", err)
	}

	state, err = hsm.ReadState()
	if err != nil {
		t.Fatal(err)
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
