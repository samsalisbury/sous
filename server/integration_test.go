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
	integrationServerTests struct {
		suite.Suite
		client restful.HTTPClient
	}

	liveServerSuite struct {
		integrationServerTests
		server *httptest.Server
	}

	inmemServerSuite struct {
		integrationServerTests
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

func (suite integrationServerTests) prepare() *graph.SousGraph {
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

func (suite *liveServerSuite) SetupTest() {
	g := suite.prepare()

	suite.server = httptest.NewServer(Handler(g, dummyLogger{}))

	var err error
	suite.integrationServerTests.client, err = restful.NewClient(suite.server.URL, dummyLogger{})
	if err != nil {
		suite.FailNow("Error constructing client: %v", err)
	}
}

func (suite *inmemServerSuite) SetupTest() {
	g := suite.prepare()
	suite.integrationServerTests.client = restful.NewInMemoryClient(Handler(g, dummyLogger{}))
}

func (suite liveServerSuite) TearDownTest() {
	suite.server.Close()
}

func (suite integrationServerTests) TestOverallRouter() {
	res, err := http.Get(suite.url + "/gdm")
	suite.NoError(err)
	gdm, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	suite.NoError(err)
	suite.Regexp(`"Deployments"`, string(gdm))
	suite.Regexp(`"Location"`, string(gdm))
	suite.NotEqual(res.Header.Get("Etag"), "")
}

func (suite integrationServerTests) decodeJSON(res *http.Response, data interface{}) {
	dec := json.NewDecoder(res.Body)
	err := dec.Decode(data)
	res.Body.Close()
	suite.NoError(err)
}

func (suite integrationServerTests) encodeJSON(data interface{}) io.Reader {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	suite.NoError(enc.Encode(data))
	return buf
}

func (suite integrationServerTests) TestMetricsPresent() {
	res, err := http.Get(suite.url + "/debug/metrics")
	suite.NoError(err)

	bs, err := ioutil.ReadAll(res.Body)
	suite.NoError(err)

	suite.Equal(string(bs), "This should be some metrics here.")
}

func (suite integrationServerTests) TestUpdateServers() {
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
	suite.Run(t, new(liveServerSuite))
}

func TestProfilingServerSuite(t *testing.T) {
	suite.Run(t, new(inmemServerSuite))
}
