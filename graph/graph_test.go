package graph

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/psyringe"
)

func TestBuildGraph(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	g := BuildGraph(ioutil.Discard, ioutil.Discard)
	g.Add(&config.Verbosity{})
	g.Add(&config.DeployFilterFlags{})
	g.Add(&config.PolicyFlags{}) //provided by SousBuild
	g.Add(&config.OTPLFlags{})   //provided by SousInit and SousDeploy

	if err := g.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}

func injectedStateManager(t *testing.T, cfg *config.Config) *StateManager {
	g := psyringe.New()
	g.Add(newStateManager)
	g.Add(LocalSousConfig{Config: cfg})

	smRcvr := struct {
		Sm *StateManager
	}{}
	err := g.Inject(&smRcvr)
	if err != nil {
		t.Fatalf("Injection err: %+v", err)
	}

	if smRcvr.Sm == nil {
		t.Fatal("StateManager not injected")
	}
	return smRcvr.Sm
}

func TestStateManagerSelectsServer(t *testing.T) {
	smgr := injectedStateManager(t, &config.Config{Server: "http://example.com"})

	if _, ok := smgr.StateManager.(*sous.HTTPStateManager); !ok {
		t.Errorf("Injected %#v which isn't a HTTPStateManager", smgr)
	}
}

func TestStateManagerSelectsGit(t *testing.T) {
	smgr := injectedStateManager(t, &config.Config{StateLocation: "/tmp/sous"})

	if _, ok := smgr.StateManager.(*storage.GitStateManager); !ok {
		t.Errorf("Injected %#v which isn't a GitStateManager", smgr)
	}
}
