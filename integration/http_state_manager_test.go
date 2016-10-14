// +build integration

package integration

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
	m := sous.Manifest{Source: sous.SourceLocation{Repo: repo}}
	m.Deployments = make(sous.DeploySpecs)
	m.Deployments[cluster] = sous.DeploySpec{Version: semv.MustParse(version)}
	return &m
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

	sm := sous.DummyStateManager{State: state}
	smm, err := sm.ReadState()
	if err != nil {
		t.Fatal("State manager double is broken", err)
	}
	if smm.Manifests.Len() <= 0 {
		t.Fatal("State manager double is empty")
	}
	gf := func() server.Injector {
		di := psyringe.New()
		di.Add(sous.NewLogSet(os.Stderr, os.Stderr, ioutil.Discard))
		graph.AddInternals(di)
		di.Add(
			func() graph.LocalStateReader { return graph.LocalStateReader{sm} },
			func() graph.LocalStateWriter { return graph.LocalStateWriter{sm} },
		)
		di.Add(&config.Verbosity{})
		return di
	}

	ts := httptest.NewServer(server.SousRouteMap.BuildRouter(gf))
	defer ts.Close()

	hsm, err := sous.NewHTTPStateManager(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	ls, err := hsm.ReadState()
	if err != nil {
		t.Error(err)
	}
	log.Printf("%#v", ls)
	if ls.Manifests.Len() != state.Manifests.Len() {
		t.Errorf("Local state has %d manifests to remote's %d", ls.Manifests.Len(), state.Manifests.Len())
	}

	ls.Manifests.Remove(diesManifest.ID())
	ls.Manifests.Add(newManifest)
	ch, there := ls.Manifests.Get(changesManifest.ID())
	if !there {
		t.Fatal("Changed manifest not in local manifests!")
	}
	chd := ch.Deployments["test-cluster"]
	chd.Version = semv.MustParse("0.18.0")
	ch.Deployments["test-cluster"] = chd
	ls.Manifests.Remove(ch.ID())
	ls.Manifests.Add(ch)
	log.Printf("%#v", ls)
	err = hsm.WriteState(ls)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	state, err = sm.ReadState()

	if ls.Manifests.Len() != state.Manifests.Len() {
		t.Errorf("After write, local state has %d manifests to remote's %d", ls.Manifests.Len(), state.Manifests.Len())
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
	if c.Deployments["test-cluster"].Version.String() != "0.18.0" {
		t.Errorf("Server's version of changed state should be '0.18.0' was %v", c.Deployments["test-cluster"].Version)
	}
}
