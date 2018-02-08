package test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
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

func (suite integrationServerTests) prepare() http.Handler {
	sourcepath, remotepath, outpath :=
		"../ext/storage/testdata/in",
		"../ext/storage/testdata/remote",
		"../ext/storage/testdata/out"

	dsm := storage.NewDiskStateManager(sourcepath)
	s, err := dsm.ReadState()
	suite.Require().NoError(err)

	storage.PrepareTestGitRepo(suite.T(), s, remotepath, outpath)

	g := graph.TestGraphWithConfig(semv.Version{}, &bytes.Buffer{}, os.Stdout, os.Stdout, "StateLocation: '"+outpath+"'\n")
	g.Add(&config.DeployFilterFlags{})
	g.Add(&config.Verbosity{})
	g.Add(graph.DryrunBoth)

	/*
		state := &sous.State{}
		state.SetEtag("qwertybeatsdvorak")
		sm := sous.DummyStateManager{State: state}

		g.Add(
			func() graph.StateReader { return graph.StateReader{StateReader: &sm} },
			func() graph.StateWriter { return graph.StateWriter{StateWriter: &sm} },
			func() *graph.StateManager { return &graph.StateManager{StateManager: &sm} },
		)
	*/

	serverScoop := struct {
		Handler graph.ServerHandler
	}{}
	g.MustInject(&serverScoop)
	if serverScoop.Handler.Handler == nil {
		suite.FailNow("Didn't inject http.Handler!")
	}
	return serverScoop.Handler.Handler
}

func (suite *liveServerSuite) SetupTest() {
	h := suite.prepare()

	suite.server = httptest.NewServer(h)
	suite.user = sous.User{}

	lt, _ := logging.NewLogSinkSpy()

	var err error
	suite.integrationServerTests.client, err = restful.NewClient(suite.server.URL, lt)
	if err != nil {
		suite.FailNow("Error constructing client: %v", err)
	}
}

func (suite *inmemServerSuite) SetupTest() {
	h := suite.prepare()
	//lt, _ := logging.NewLogSinkSpy()
	lt := logging.NewLogSet(semv.MustParse("0.0.0"), "", "", os.Stdout)
	lt.BeChatty()
	suite.user = sous.User{}
	var err error
	suite.integrationServerTests.client, err = restful.NewInMemoryClient(h, lt)
	if err != nil {
		suite.FailNow("Error constructing client: %v", err)
	}
}

func (suite liveServerSuite) TearDownTest() {
	suite.server.Close()
}

func (suite integrationServerTests) TestOverallRouter() {

	gdm := server.GDMWrapper{}
	updater, err := suite.client.Retrieve("./gdm", nil, &gdm, suite.user.HTTPHeaders())
	suite.NoError(err)

	suite.Len(gdm.Deployments, 2)
	suite.NotZero(updater)
}

func (suite integrationServerTests) TestUpdateServers() {
	data := server.ServerListData{}
	updater, err := suite.client.Retrieve("./servers", nil, &data, nil)

	suite.NoError(err)
	suite.Len(data.Servers, 0)

	newServers := server.ServerListData{
		Servers: []server.NameData{{ClusterName: "name", URL: "http://url"}},
	}

	err = updater.Update(&newServers, nil)
	suite.NoError(err)

	data = server.ServerListData{}
	_, err = suite.client.Retrieve("./servers", nil, &data, nil)
	suite.NoError(err)
	suite.Len(data.Servers, 1)
}

func (suite integrationServerTests) TestUpdateStateDeployments() {
	data := server.GDMWrapper{Deployments: []*sous.Deployment{}}
	err := suite.client.Create("./state/deployments", nil, &data, nil)
	suite.Error(err, `412 Precondition Failed: "resource present for If-None-Match=*!\n"`)

	updater, err := suite.client.Retrieve("./state/deployments", nil, &data, nil)
	suite.NoError(err)
	suite.Equal(data, "a rabbit")
	suite.NotNil(updater)
}

func TestLiveServerSuite(t *testing.T) {
	suite.Run(t, new(liveServerSuite))
}

func TestInMemoryServerSuite(t *testing.T) {
	suite.Run(t, new(inmemServerSuite))
}
