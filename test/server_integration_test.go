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
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/suite"
)

type (
	integrationServerTests struct {
		suite.Suite
		client restful.HTTPClient
		user   sous.User
		log    logging.LogSinkController
	}

	liveServerSuite struct {
		integrationServerTests
		server *httptest.Server
	}

	inmemServerSuite struct {
		integrationServerTests
	}
)

func (suite integrationServerTests) prepare() (logging.LogSink, http.Handler) {
	sourcepath, remotepath, outpath :=
		"../ext/storage/testdata/in",
		"../ext/storage/testdata/remote",
		"../ext/storage/testdata/out"

	dsm := storage.NewDiskStateManager(sourcepath)
	s, err := dsm.ReadState()
	suite.Require().NoError(err)

	storage.PrepareTestGitRepo(suite.T(), s, remotepath, outpath)

	log, ctrl := logging.NewLogSinkSpy()
	suite.log = ctrl

	g := graph.TestGraphWithConfig(semv.Version{}, &bytes.Buffer{}, os.Stdout, os.Stdout, "StateLocation: '"+outpath+"'\n")
	g.Add(&config.DeployFilterFlags{})
	g.Add(&config.Verbosity{})
	g.Add(graph.DryrunBoth)

	testGraph := psyringe.TestPsyringe{Psyringe: g.Psyringe}
	testGraph.Replace(graph.LogSink{LogSink: log})
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
	return log, serverScoop.Handler.Handler
}

func (suite *liveServerSuite) SetupTest() {
	lt, h := suite.prepare()

	suite.server = httptest.NewServer(h)
	suite.user = sous.User{}

	var err error
	suite.integrationServerTests.client, err = restful.NewClient(suite.server.URL, lt)
	if err != nil {
		suite.FailNow("Error constructing client: %v", err)
	}
}

func (suite *inmemServerSuite) SetupTest() {
	lt, h := suite.prepare()

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

func (suite integrationServerTests) TestUpdateStateDeployments_Precondition() {
	data := server.GDMWrapper{Deployments: []*sous.Deployment{}}
	err := suite.client.Create("./state/deployments", nil, &data, nil)
	suite.Error(err, `412 Precondition Failed: "resource present for If-None-Match=*!\n"`)
}

func (suite integrationServerTests) TestUpdateStateDeployments() {
	data := server.GDMWrapper{Deployments: []*sous.Deployment{}}
	updater, err := suite.client.Retrieve("./state/deployments", nil, &data, nil)
	suite.log.DumpLogs(suite.T())
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
