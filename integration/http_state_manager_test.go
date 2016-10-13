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
)

func TestWriteState(t *testing.T) {
	state := &sous.State{}
	state.Manifests = sous.NewManifests()
	state.Manifests.Add(&sous.Manifest{})
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

	// local HSM, pointed at the httptest.Server wrapping same
	// ReadState -> 3 distinct manifests (by repo)
	// Remove one, change one, add one
	// WriteState
	// Confirm: one manifest unchanged, one gone, one changed, new one added
}
