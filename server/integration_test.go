package server

import (
	"bytes"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/suite"
)

type (
	integrationServerTests struct {
		suite.Suite
		client restful.HTTPClient
		user   sous.User
	}

	liveServerSuite struct {
		integrationServerTests
		server *httptest.Server
	}

	inmemServerSuite struct {
		integrationServerTests
	}
)

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
	suite.user = sous.User{}

	var err error
	suite.integrationServerTests.client, err = restful.NewClient(suite.server.URL, dummyLogger{})
	if err != nil {
		suite.FailNow("Error constructing client: %v", err)
	}
}

func (suite *inmemServerSuite) SetupTest() {
	g := suite.prepare()

	suite.user = sous.User{}
	var err error
	suite.integrationServerTests.client, err = restful.NewInMemoryClient(Handler(g, dummyLogger{}), dummyLogger{})
	if err != nil {
		suite.FailNow("Error constructing client: %v", err)
	}
}

func (suite liveServerSuite) TearDownTest() {
	suite.server.Close()
}

func (suite integrationServerTests) TestOverallRouter() {

	gdm := gdmWrapper{}
	updater, err := suite.client.Retrieve("./gdm", nil, &gdm, suite.user.HTTPHeaders())
	suite.NoError(err)

	suite.Len(gdm.Deployments, 2)
	suite.NotZero(updater)
}

func (suite integrationServerTests) TestUpdateServers() {
	data := serverListData{}
	updater, err := suite.client.Retrieve("./servers", nil, &data, nil)

	suite.NoError(err)
	suite.Len(data.Servers, 0)

	newServers := serverListData{
		Servers: []server{server{ClusterName: "name", URL: "url"}},
	}

	err = updater.Update(nil, &newServers, nil)
	suite.NoError(err)

	data = serverListData{}
	_, err = suite.client.Retrieve("./servers", nil, &data, nil)
	suite.NoError(err)
	suite.Len(data.Servers, 1)
}

func TestLiveServerSuite(t *testing.T) {
	suite.Run(t, new(liveServerSuite))
}

func TestInMemoryServerSuite(t *testing.T) {
	suite.Run(t, new(inmemServerSuite))
}
