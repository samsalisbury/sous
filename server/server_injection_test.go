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
)

func basicInjectedHandler(factory ExchangeFactory, t *testing.T) Exchanger {
	require := require.New(t)

	gf := func() Injector {
		g := graph.TestGraphWithConfig(&bytes.Buffer{}, os.Stdout, os.Stdout, "StateLocation: '../ext/storage/testdata/in'\n")
		g.Add(&config.Verbosity{})
		return g
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
