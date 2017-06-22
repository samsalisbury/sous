package graph

import (
	"bytes"
	"io/ioutil"
	"log"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/psyringe"
)

func TestBuildGraph(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	g := BuildGraph(&bytes.Buffer{}, ioutil.Discard, ioutil.Discard)
	g.Add(DryrunBoth)
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
	g.Add(newUser)
	g.Add(sous.SilentLogSet())
	g.Add(newStateManager)
	g.Add(LocalSousConfig{Config: cfg})
	g.Add(newHTTPClient)

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
	smgr := injectedStateManager(t, &config.Config{Server: "http://example.com", StateLocation: "/tmp/sous"})

	if _, ok := smgr.StateManager.(*sous.HTTPStateManager); !ok {
		t.Errorf("Injected %#v which isn't a HTTPStateManager", smgr.StateManager)
	}
}

func TestStateManagerSelectsGit(t *testing.T) {
	smgr := injectedStateManager(t, &config.Config{Server: "", StateLocation: "/tmp/sous"})

	if _, ok := smgr.StateManager.(*storage.GitStateManager); !ok {
		t.Errorf("Injected %#v which isn't a GitStateManager", smgr.StateManager)
	}
}

func testBuildInserter(t *testing.T, serverStr string) sous.Inserter {
	ins, err := newInserter(LocalSousConfig{Config: &config.Config{
		Server: serverStr,
		Docker: docker.Config{
			DatabaseDriver:     "sqlite3_sous",
			DatabaseConnection: docker.InMemory,
		},
	}}, LocalDockerClient{})
	if err != nil {
		t.Fatal(err)
	}
	return ins
}

func TestNameInserterSelectsNameCache(t *testing.T) {
	ins := testBuildInserter(t, "")
	if _, ok := ins.(*docker.NameCache); !ok {
		t.Errorf("Injected %#v which isn't a docker.NameCache", ins)
	}
}

func TestNameInserterSelectsHTTP(t *testing.T) {
	ins := testBuildInserter(t, "http//example.com")
	if _, ok := ins.(*sous.HTTPNameInserter); !ok {
		t.Errorf("Injected %#v which isn't a docker.NameCache", ins)
	}
}

func TestNewBuildConfig(t *testing.T) {
	f := &config.DeployFilterFlags{}
	p := &config.PolicyFlags{}
	bc := &sous.BuildContext{
		Sh: &shell.Sh{},
		Source: sous.SourceContext{
			RemoteURL: "github.com/opentable/present",
			RemoteURLs: []string{
				"github.com/opentable/present",
				"github.com/opentable/also",
			},
			Revision:           "abcdef",
			NearestTagName:     "1.2.3",
			NearestTagRevision: "abcdef",
			Tags: []sous.Tag{
				sous.Tag{Name: "1.2.3"},
			},
		},
	}

	cfg := newBuildConfig(f, p, bc)
	if cfg.Tag != `1.2.3` {
		t.Errorf("Build config's tag wasn't 1.2.3: %#v", cfg.Tag)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Not valid build config: %+v", err)
	}

}
