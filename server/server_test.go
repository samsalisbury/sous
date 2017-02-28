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

	"github.com/nyarly/testify/suite"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
)

type serverSuite struct {
	suite.Suite
	server *httptest.Server
	url    string
}

func (suite *serverSuite) SetupTest() {
	g := graph.TestGraphWithConfig(&bytes.Buffer{}, os.Stdout, os.Stdout,
		"StateLocation: '../ext/storage/testdata/in'\n")
	g.Add(&config.Verbosity{})
	suite.server = httptest.NewServer(Handler(g))
	suite.url = suite.server.URL
}

func (suite *serverSuite) TearDownTest() {
	suite.server.Close()
	suite.server = nil
	suite.url = ""
}

func (suite *serverSuite) TestOverallRouter() {
	res, err := http.Get(suite.url + "/gdm")
	suite.NoError(err)
	gdm, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	suite.NoError(err)
	suite.Regexp(`"Deployments"`, string(gdm))
	suite.Regexp(`"Location"`, string(gdm))
	suite.NotEqual(res.Header.Get("Etag"), "")
}

func (suite *serverSuite) decodeJSON(res *http.Response, data interface{}) {
	dec := json.NewDecoder(res.Body)
	err := dec.Decode(data)
	res.Body.Close()
	suite.NoError(err)
}

func (suite *serverSuite) encodeJSON(data interface{}) io.Reader {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	suite.NoError(enc.Encode(data))
	return buf
}

func (suite *serverSuite) TestUpdateServers() {
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

	req, err := http.NewRequest("PUT", suite.url+"/servers", suite.encodeJSON(newServers))
	req.Header.Set("If-Match", etag)
	suite.NoError(err)
	res, err = client.Do(req)
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
