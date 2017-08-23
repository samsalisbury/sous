package graph

import (
	"bytes"
	"os"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func basicInjectedHandler(factory restful.ExchangeFactory, t *testing.T) restful.Exchanger {
	emptyState := sous.State{
		Manifests: sous.NewManifests(),
		Defs:      sous.Defs{DockerRepo: "nowhere.example.com"},
	}
	storage.PrepareTestGitRepo(t, &emptyState, "../ext/storage/testdata/remote", "../ext/storage/testdata/out")
	g := TestGraphWithConfig(&bytes.Buffer{}, os.Stdout, os.Stdout, "StateLocation: '../ext/storage/testdata/out'\n")

	g.Add(&config.Verbosity{})
	g.Add(&config.DeployFilterFlags{Cluster: "test"})
	g.Add(DryrunBoth)

	gf := func() restful.Injector {
		return g.Clone()
	}

	exchLogger := server.SousRouteMap.SingleExchanger(factory, gf, restful.PlaceholderLogger())

	logger, ok := exchLogger.(*restful.ExchangeLogger)
	require.True(t, ok)

	return logger.Exchanger
}

func TestServerListHandlerInjection(t *testing.T) {
	slr := &server.ServerListResource{}
	slh := basicInjectedHandler(slr.Get, t)

	serverListGet, ok := slh.(*server.ServerListHandler)
	require.True(t, ok)

	assert.NotNil(t, serverListGet.Config)
}

func TestServerGetDefHandlerInjection(t *testing.T) {
	sdr := &server.StateDefResource{}

	slh := basicInjectedHandler(sdr.Get, t)

	serverDefsGet, ok := slh.(*server.StateDefGetHandler)
	require.True(t, ok)

	assert.NotNil(t, serverDefsGet.State)
}

func TestServerGetManifestHandlerInjection(t *testing.T) {
	mr := &server.ManifestResource{}

	mgh := basicInjectedHandler(mr.Get, t)

	serverManifestGet, ok := mgh.(*server.GETManifestHandler)
	require.True(t, ok)

	assert.NotNil(t, serverManifestGet.State)
}

func TestServerPutGDMHandlerInjection(t *testing.T) {
	sdr := &server.GDMResource{}

	slh := basicInjectedHandler(sdr.Put, t)

	handler, ok := slh.(*server.PUTGDMHandler)
	require.True(t, ok)

	assert.NotNil(t, handler.StateManager.StateManager)
}

func TestStatusHandlerInjection(t *testing.T) {
	sr := &server.StatusResource{}
	sh := basicInjectedHandler(sr.Get, t)

	statusGet, ok := sh.(*server.StatusHandler)
	require.True(t, ok)

	logging.Log.Debug.Printf("%#v", statusGet)
	assert.NotPanics(t, func() {
		_ = statusGet.AutoResolver.String()
	})
}

func TestGDMHandlerInjection(t *testing.T) {
	sr := &server.GDMResource{}
	sh := basicInjectedHandler(sr.Put, t)

	gdmPut, ok := sh.(*server.PUTGDMHandler)
	require.True(t, ok)

	t.Logf("%#v", gdmPut)
	assert.NotNil(t, gdmPut.StateManager.StateManager)
}
