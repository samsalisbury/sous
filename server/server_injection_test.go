package server

import (
	"bytes"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/nyarly/testify/assert"
	"github.com/nyarly/testify/require"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
)

func basicInjectedHandler(factory ExchangeFactory, t *testing.T) Exchanger {
	require := require.New(t)

	g := graph.TestGraphWithConfig(&bytes.Buffer{}, os.Stdout, os.Stdout, "StateLocation: '../ext/storage/testdata/in'\n")
	g.Add(&config.Verbosity{})
	g.Add(&config.DeployFilterFlags{Cluster: "test"})
	g.Add(graph.DryrunBoth)

	gf := func() Injector {
		return g.Clone()
	}

	r := httprouter.New()
	mh := SousRouteMap.buildMetaHandler(r, gf)

	w := httptest.NewRecorder()
	rq := httptest.NewRequest("GET", "/", nil)

	serverListGetLogger := mh.injectedHandler(factory, w, rq, httprouter.Params{})

	logger, ok := serverListGetLogger.(*ExchangeLogger)
	require.True(ok)

	return logger.Exchanger
}

func TestServerListHandlerInjection(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	slr := &ServerListResource{}
	slh := basicInjectedHandler(slr.Get, t)

	serverListGet, ok := slh.(*ServerListHandler)
	require.True(ok)

	assert.NotNil(serverListGet.Config)
}

func TestStatusHandlerInjection(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	sr := &StatusResource{}
	sh := basicInjectedHandler(sr.Get, t)

	statusGet, ok := sh.(*StatusHandler)
	require.True(ok)

	sous.Log.Debug.Printf("%#v", statusGet)
	assert.NotPanics(func() {
		statusGet.AutoResolver.String()
	})
}
