package test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/suite"
)

type (
	serverTests struct {
		suite.Suite
		server *httptest.Server
		url    string
	}

	serverSuite struct {
		serverTests
	}

	profServerSuite struct {
		serverTests
	}

	dummyLogger struct{}
)

func (dummyLogger) ExpHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This should be some metrics here."))
	})
}

func (suite serverTests) prepare(extras ...interface{}) http.Handler {
	sourcepath, remotepath, outpath :=
		"../ext/storage/testdata/in",
		"../ext/storage/testdata/remote",
		"../ext/storage/testdata/out"

	dsm := storage.NewDiskStateManager(sourcepath, logging.SilentLogSet())
	s, err := dsm.ReadState()
	suite.Require().NoError(err)

	storage.PrepareTestGitRepo(suite.T(), s, remotepath, outpath)

	g := graph.TestGraphWithConfig(suite.T(), semv.Version{}, &bytes.Buffer{}, os.Stdout, os.Stdout, "StateLocation: '"+outpath+"'\n")
	g.Add(extras...)

	dff := config.DeployFilterFlags{}
	dff.Cluster = "test"
	g.Add(&config.Verbosity{})
	g.Add(&dff)
	g.Add(graph.DryrunBoth)

	serverScoop := struct {
		Handler graph.ServerHandler
	}{}

	g.MustInject(&serverScoop)
	if serverScoop.Handler.Handler == nil {
		suite.FailNow("Didn't inject http.Handler!")
	}
	return serverScoop.Handler.Handler
}

func (suite *serverSuite) SetupTest() {
	h := suite.prepare()

	suite.serverTests.server = httptest.NewServer(h)
	suite.serverTests.url = suite.server.URL
}

func (suite *profServerSuite) SetupTest() {
	h := suite.prepare(graph.ProfilingServer(true))
	suite.serverTests.server = httptest.NewServer(h) // <--- profiling
	suite.serverTests.url = suite.server.URL
}

func (suite *profServerSuite) TestDebugPprof() {
	res, err := http.Get(suite.url + "/debug/pprof/")
	suite.NoError(err)
	res.Body.Close()
}

func (suite serverTests) TearDownTest() {
	suite.server.Close()
}

func (suite serverTests) TestOverallRouter() {
	client := http.Client{}
	req, err := http.NewRequest("GET", suite.url+"/gdm", nil)
	suite.Require().NoError(err)
	req.Header.Set("X-Gatelatch", "yes")
	res, err := client.Do(req)
	suite.NoError(err)
	gdm, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	suite.NoError(err)
	suite.Regexp(`"Deployments"`, string(gdm))
	suite.Regexp(`"Location"`, string(gdm))
	suite.NotEqual(res.Header.Get("Etag"), "")
}

func (suite serverTests) decodeJSON(res *http.Response, data interface{}) {
	dec := json.NewDecoder(res.Body)
	err := dec.Decode(data)
	res.Body.Close()
	suite.NoError(err)
}

func (suite serverTests) encodeJSON(data interface{}) io.Reader {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	suite.NoError(enc.Encode(data))
	return buf
}

func (suite serverTests) TestMetricsPresent() {
	res, err := http.Get(suite.url + "/debug/metrics")
	suite.NoError(err)

	bs, err := ioutil.ReadAll(res.Body)
	suite.NoError(err)

	suite.Regexp("Mallocs", string(bs))
}

func (suite serverTests) TestUpdateServers() {
	res, err := http.Get(suite.url + "/servers")
	suite.NoError(err)

	etag := res.Header.Get("Etag")
	data := &server.ServerListData{}
	suite.decodeJSON(res, data)
	suite.Len(data.Servers, 0)

	client := &http.Client{}

	newServers := &server.ServerListData{
		Servers: []server.NameData{{ClusterName: "name", URL: "http://url"}},
	}

	req, err := http.NewRequest("PUT", suite.url+"/servers", restful.InjectCanaryAttr(suite.encodeJSON(newServers), etag))
	req.Header.Set("If-Match", etag)
	suite.NoError(err)
	_, err = client.Do(req)
	suite.NoError(err)

	res, err = http.Get(suite.url + "/servers")
	suite.NoError(err)

	data = &server.ServerListData{}
	suite.decodeJSON(res, data)
	suite.Len(data.Servers, 1)
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(serverSuite))
}

func TestProfilingServerSuite(t *testing.T) {
	suite.Run(t, new(profServerSuite))
}
