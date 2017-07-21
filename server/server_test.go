package server

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
	"github.com/opentable/sous/util/restful"
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

func (dummyLogger) Warnf(string, ...interface{})  {}
func (dummyLogger) Debugf(string, ...interface{}) {}
func (dummyLogger) Vomitf(string, ...interface{}) {}
func (dummyLogger) ExpHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This should be some metrics here."))
	})
}

func (suite serverTests) prepare() *graph.SousGraph {
	sourcepath, remotepath, outpath :=
		"../ext/storage/testdata/in",
		"../ext/storage/testdata/remote",
		"../ext/storage/testdata/out"

	dsm := storage.NewDiskStateManager(sourcepath)
	s, err := dsm.ReadState()
	suite.Require().NoError(err)

	storage.PrepareTestGitRepo(suite.T(), s, remotepath, outpath)

	g := graph.TestGraphWithConfig(&bytes.Buffer{}, os.Stdout, os.Stdout,
		"StateLocation: '"+outpath+"'\n")
	g.Add(&config.Verbosity{})
	return g
}

func (suite *serverSuite) SetupTest() {
	g := suite.prepare()

	suite.serverTests.server = httptest.NewServer(Handler(g, dummyLogger{}))
	suite.serverTests.url = suite.server.URL
}

func (suite *profServerSuite) SetupTest() {
	g := suite.prepare()
	suite.serverTests.server = httptest.NewServer(ProfilingHandler(g, dummyLogger{})) // <--- profiling
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
	res, err := http.Get(suite.url + "/gdm")
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

	suite.Equal(string(bs), "This should be some metrics here.")
}

func (suite serverTests) TestUpdateServers() {
	res, err := http.Get(suite.url + "/servers")
	suite.NoError(err)

	etag := res.Header.Get("Etag")
	data := &serverListData{}
	suite.decodeJSON(res, data)
	suite.Len(data.Servers, 0)

	client := &http.Client{}

	newServers := &serverListData{
		Servers: []server{server{ClusterName: "name", URL: "url"}},
	}

	req, err := http.NewRequest("PUT", suite.url+"/servers", restful.InjectCanaryAttr(suite.encodeJSON(newServers), etag))
	req.Header.Set("If-Match", etag)
	suite.NoError(err)
	_, err = client.Do(req)
	suite.NoError(err)

	res, err = http.Get(suite.url + "/servers")
	suite.NoError(err)

	data = &serverListData{}
	suite.decodeJSON(res, data)
	suite.Len(data.Servers, 1)
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(serverSuite))
}

func TestProfilingServerSuite(t *testing.T) {
	suite.Run(t, new(profServerSuite))
}
