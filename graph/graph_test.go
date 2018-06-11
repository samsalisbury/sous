package graph

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatusPoller(t *testing.T) {
	testPoller := func(sf config.DeployFilterFlags) *sous.StatusPoller {
		shc := sous.SourceHostChooser{}
		f, err := sf.BuildFilter(shc.ParseSourceLocation)
		require.NoError(t, err)

		// func newRefinedResolveFilter(f *sous.ResolveFilter, discovered *SourceContextDiscovery) (*RefinedResolveFilter, error) {

		disc := &SourceContextDiscovery{
			SourceContext: &sous.SourceContext{
				PrimaryRemoteURL: "github.com/somewhere/thing",
				NearestTag:       sous.Tag{Name: "1.2.3", Revision: "cabbage"},
				Tags:             []sous.Tag{},
			},
		}
		rf, err := newRefinedResolveFilter(f, disc)
		require.NoError(t, err)
		cl := newDummyHTTPClient()
		user := sous.User{}

		//newStatusPoller(cl HTTPClient, rf *RefinedResolveFilter, user sous.User) *sous.StatusPoller {
		return newStatusPoller(cl, rf, user, LogSink{logging.SilentLogSet()})
	}

	p := testPoller(config.DeployFilterFlags{})
	assert.False(t, p.ResolveFilter.Flavor.All())
}

func TestBuildGraph(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	g := BuildGraph(semv.MustParse("0.0.0"), &bytes.Buffer{}, ioutil.Discard, ioutil.Discard)

	g.Add(DryrunBoth)
	g.Add(&config.Verbosity{})
	g.Add(&config.DeployFilterFlags{})
	g.Add(&config.PolicyFlags{}) //provided by SousBuild
	g.Add(&config.OTPLFlags{})   //provided by SousInit and SousDeploy
	g.Add()

	if err := g.Test(); err != nil {
		t.Fatalf("invalid graph: %s", err)
	}
}

func TestLogSink(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	g := BuildGraph(semv.MustParse("0.0.0"), &bytes.Buffer{}, ioutil.Discard, ioutil.Discard)

	g.Add(&config.Verbosity{})

	tg := &psyringe.TestPsyringe{Psyringe: g.Psyringe}
	rawConfig := RawConfig{Config: &config.Config{}}
	logcfg := &rawConfig.Config.Logging
	logcfg.Basic.Level = "debug"
	//logcfg.Kafka.Enabled = true
	logcfg.Kafka.DefaultLevel = "debug"
	logcfg.Kafka.Topic = "logging"
	logcfg.Kafka.BrokerList = "kafka.example.com:9292"
	logcfg.Graphite.Enabled = true
	logcfg.Graphite.Server = "localhost:3333"

	tg.Replace(rawConfig)

	scoop := struct{ LogSink }{}

	tg.MustInject(&scoop)

	set, is := scoop.LogSink.LogSink.(*logging.LogSet)

	assert.True(t, is)
	assert.NoError(t, logging.AssertConfiguration(set, "localhost:3333"))
}

func TestComponentLocatorInjection(t *testing.T) {
	g := BuildGraph(semv.MustParse("2.3.7-somenonsense+ignorebuilds"), &bytes.Buffer{}, ioutil.Discard, ioutil.Discard)

	g.Add(DryrunBoth)
	g.Add(&config.Verbosity{})
	g.Add(&config.DeployFilterFlags{Cluster: "test"})

	tg := &psyringe.TestPsyringe{Psyringe: g.Psyringe}
	rawConfig := RawConfig{Config: &config.Config{}}
	logcfg := &rawConfig.Config.Logging
	logcfg.Basic.Level = "debug"
	logcfg.Kafka.DefaultLevel = "debug"
	logcfg.Kafka.Topic = "logging"
	logcfg.Kafka.BrokerList = "kafka.example.com:9292"
	logcfg.Graphite.Enabled = true
	logcfg.Graphite.Server = "localhost:3333"

	port := "6543"
	if ps, got := os.LookupEnv("PGPORT"); got {
		port = ps
	}

	name := "componentlocatorinjection"
	sous.SetupDB(t)
	defer sous.ReleaseDB(t)

	host := "localhost"
	if h, got := os.LookupEnv("PGHOST"); got {
		host = h
	}

	rawConfig.Database.Host = host
	rawConfig.Database.Port = port
	rawConfig.Database.DBName = "sous_test_" + name

	tg.Replace(rawConfig)
	tg.Replace(func() serverInserter {
		ins, _ := sous.NewInserterSpy()

		return serverInserter{ins}
	})

	scoop := struct{ server.ComponentLocator }{}

	tg.MustInject(&scoop)

	locator := scoop.ComponentLocator

	assert.NotNil(t, locator.LogSink)
	assert.NotNil(t, locator.Config)
	assert.NotNil(t, locator.Inserter)
	assert.NotNil(t, locator.StateManager)
	assert.NotNil(t, locator.ResolveFilter)
	assert.NotNil(t, locator.AutoResolver)
	assert.Equal(t, locator.Version.Format("M.m.p"), "2.3.7")
}

/*
Nixing these tests: server/client state managers should be selected based on `sous server` or not.
func TestStateManagerSelectsServer(t *testing.T) {
	smgr := injectedStateManager(t, &config.Config{Server: "http://example.com", StateLocation: "/tmp/sous"})

	if _, ok := smgr.StateManager.(*sous.HTTPStateManager); !ok {
		t.Errorf("Injected %#v which isn't a HTTPStateManager", smgr.StateManager)
	}
}

func TestStateManagerSelectsDuplex(t *testing.T) {
	smgr := injectedStateManager(t, &config.Config{Server: "", StateLocation: "/tmp/sous"})

	_, ok := smgr.StateManager.(*storage.DuplexStateManager)
	if !ok {
		t.Errorf("Injected %#v which isn't a DuplexStateManager", smgr.StateManager)
	}
}
*/

var silentLogSink = DefaultLogSink{LogSink: nonDefaultSilentLogSink}

var nonDefaultSilentLogSink = LogSink{LogSink: logging.SilentLogSet()}

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

	cfg := newBuildConfig(nonDefaultSilentLogSink, f, p, bc)
	if cfg.Tag != `1.2.3` {
		t.Errorf("Build config's tag wasn't 1.2.3: %#v", cfg.Tag)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Not valid build config: %+v", err)
	}

}
