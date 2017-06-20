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
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/suite"
)

type serverTests struct {
	suite.Suite
	server *httptest.Server
	url    string
}

type serverSuite struct {
	serverTests
}

type profServerSuite struct {
	serverTests
}

func (suite *serverSuite) SetupTest() {
	g := graph.TestGraphWithConfig(&bytes.Buffer{}, os.Stdout, os.Stdout,
		"StateLocation: '../ext/storage/testdata/in'\n")
	g.Add(&config.Verbosity{})
	suite.serverTests.server = httptest.NewServer(Handler(g))
	suite.serverTests.url = suite.server.URL
}

func (suite *profServerSuite) SetupTest() {
	g := graph.TestGraphWithConfig(&bytes.Buffer{}, os.Stdout, os.Stdout,
		"StateLocation: '../ext/storage/testdata/in'\n")
	g.Add(&config.Verbosity{})
	suite.serverTests.server = httptest.NewServer(profilingHandler(g)) // <--- profiling
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
