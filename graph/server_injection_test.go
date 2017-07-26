package graph

import (
	"bytes"
	"os"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func basicInjectedHandler(factory restful.ExchangeFactory, t *testing.T) restful.Exchanger {
	g := TestGraphWithConfig(&bytes.Buffer{}, os.Stdout, os.Stdout, "StateLocation: '../ext/storage/testdata/in'\n")
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
