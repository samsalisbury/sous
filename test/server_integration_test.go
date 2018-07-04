package test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/dto"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/pborman/uuid"
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/suite"
)

type (
	integrationServerTests struct {
		suite.Suite
		client           restful.HTTPClient
		user             sous.User
		log              logging.LogSinkController
		componentLocator server.ComponentLocator
	}

	liveServerSuite struct {
		integrationServerTests
		server  *httptest.Server
		cleanup func()
	}

	inmemServerSuite struct {
		integrationServerTests
		cleanup func()
	}
)

// prepare returns a logging.LogSink and http.Handler for use in tests.
// It also returns a cleanup function which should be called to remove
// temp files created after each test run.
func (suite *integrationServerTests) prepare() (logging.LogSink, http.Handler, func()) {
	td, err := filepath.Abs("../ext/storage/testdata")
	if err != nil {
		suite.FailNow("setup failed: %s", err)
	}
	temp := filepath.Join(os.TempDir(), "soustests", uuid.New())
	sourcepath, remotepath, outpath :=
		filepath.Join(td, "in"),
		filepath.Join(temp, "remote"),
		filepath.Join(temp, "out")

	dsm := storage.NewDiskStateManager(sourcepath, logging.SilentLogSet())
	s, err := dsm.ReadState()
	suite.Require().NoError(err)

	storage.PrepareTestGitRepo(suite.T(), s, remotepath, outpath)

	log, ctrl := logging.NewLogSinkSpy()
	suite.log = ctrl

	dff := config.DeployFilterFlags{}
	dff.Cluster = "cluster-1"

	g := graph.TestGraphWithConfig(suite.T(), semv.Version{}, &bytes.Buffer{}, os.Stdout, os.Stdout, "StateLocation: '"+outpath+"'\n")
	g.Add(&config.Verbosity{})
	g.Add(&dff)
	g.Add(graph.DryrunBoth)

	testGraph := psyringe.TestPsyringe{Psyringe: g.Psyringe}
	testGraph.Replace(graph.LogSink{LogSink: log})

	serverScoop := struct {
		ComponentLocator server.ComponentLocator
	}{}

	g.MustInject(&serverScoop)

	suite.componentLocator = serverScoop.ComponentLocator

	// Replace the default queueset with this one that doesn't process anything.
	suite.componentLocator.QueueSet = sous.NewR11nQueueSet()
	testGraph.Replace(suite.componentLocator)

	handlerScoop := struct {
		Handler graph.ServerHandler
	}{}
	g.MustInject(&handlerScoop)
	return log, handlerScoop.Handler.Handler, func() {
		if err := os.RemoveAll(outpath); err != nil {
			suite.T().Errorf("cleanup failed: %s", err)
		}
		if err := os.RemoveAll(remotepath); err != nil {
			suite.T().Errorf("cleanup failed: %s", err)
		}
	}

}

func (suite integrationServerTests) errorMatches(err error, regexp string) {
	suite.T().Helper()
	if suite.Error(err) {
		suite.Regexp(regexp, err.Error())
	}
}

func (suite *liveServerSuite) SetupTest() {
	lt, h, cleanup := suite.prepare()
	suite.cleanup = cleanup

	suite.server = httptest.NewServer(h)
	suite.user = sous.User{}

	var err error
	suite.integrationServerTests.client, err = restful.NewClient(suite.server.URL, lt)
	if err != nil {
		suite.FailNow("Error constructing client: %v", err)
	}
}

func (suite *inmemServerSuite) SetupTest() {
	lt, h, cleanup := suite.prepare()
	suite.cleanup = cleanup

	suite.user = sous.User{}
	var err error
	suite.integrationServerTests.client, err = restful.NewInMemoryClient(h, lt)
	if err != nil {
		suite.FailNow("Error constructing client: %v", err)
	}
}

func (suite liveServerSuite) TearDownTest() {
	suite.server.Close()
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func (suite inmemServerSuite) TearDownTest() {
	if suite.cleanup != nil {
		suite.cleanup()
	}
}

func (suite integrationServerTests) TestOverallRouter() {

	gdm := dto.GDMWrapper{}
	updater, err := suite.client.Retrieve("./gdm", nil, &gdm, suite.user.HTTPHeaders())
	suite.NoError(err)

	suite.Len(gdm.Deployments, 4)
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

	_, err = updater.Update(&newServers, nil)
	suite.NoError(err)

	data = server.ServerListData{}
	_, err = suite.client.Retrieve("./servers", nil, &data, nil)
	suite.NoError(err)
	suite.Len(data.Servers, 1)
}

func (suite integrationServerTests) TestUpdateStateDeployments_Precondition() {
	data := dto.GDMWrapper{Deployments: []*sous.Deployment{}}
	res, err := suite.client.Create("./state/deployments", nil, &data, nil)
	suite.errorMatches(err, `^Create \./state/deployments params: map\[\]: 412 Precondition Failed: resource present for If-None-Match=\*!`)
	suite.Nil(res)
}

func (suite integrationServerTests) assertNoEmptySingReqIDs(name string, gdm dto.GDMWrapper) {
	for _, d := range gdm.Deployments {
		if d.DeployConfig.SingularityRequestID == "" {
			suite.FailNow("%s contains empty sing req id", name)
		}
	}
}

func (suite integrationServerTests) TestUpdateStateDeployments_Update() {
	data := dto.GDMWrapper{}

	updater, err := suite.client.Retrieve("./state/deployments", nil, &data, nil)
	suite.NoError(err)
	suite.Len(data.Deployments, 2)
	suite.NotNil(updater)

	suite.assertNoEmptySingReqIDs("test data", data)

	data.Deployments = append(data.Deployments, sous.DeploymentFixture("sequenced-repo"))

	suite.assertNoEmptySingReqIDs("updated test data", data)

	_, err = updater.Update(&data, nil)
	suite.NoError(err)

	_, err = suite.client.Retrieve("./state/deployments", nil, &data, nil)
	suite.NoError(err)
	suite.Len(data.Deployments, 3)
}

func (suite integrationServerTests) TestPUTSingleDeployment() {
	params := map[string]string{
		"cluster": "no-such-place",
		"repo":    "github.com/xxx/xxx",
		"offset":  "",
		"tag":     "1.0.1",
	}
	rez, err := suite.client.Retrieve("/single-deployment", params, nil, nil)
	suite.errorMatches(err, `404 Not Found.*No manifest with ID`) // empty ID gets 404
	suite.Nil(rez)

	params = map[string]string{
		"cluster": "cluster-1",
		"repo":    "github.com/opentable/sous",
		"offset":  "",
		"tag":     "1.0.1",
		"force":   "false",
	}
	data := server.SingleDeploymentBody{}
	rez, err = suite.client.Retrieve("/single-deployment", params, &data, nil)
	suite.NoError(err)
	suite.NotNil(rez)
	data.Deployment.NumInstances = 100
	updater, err := rez.Update(&data, nil)
	suite.NoError(err)
	suite.Regexp(`deploy-queue-item\?.*action=`, updater.Location())
}

func (suite integrationServerTests) TestGetAllDeployQueues_empty() {
	data := server.DeploymentQueuesResponse{}
	updater, err := suite.client.Retrieve("./all-deploy-queues", nil, &data, nil)
	suite.NoError(err)
	suite.Len(data.Queues, 0)
	suite.NotNil(updater)
}

func (suite integrationServerTests) TestGetAllDeployQueues_nonempty() {
	// Since componentLocator is not a pointer, we need to replace the QueueSet
	// in memory directly.
	// Go vet complains on the next about copying a sync.RWMutex.
	// In this case it's a zero RWMutex at the time of copy, so it is in fact
	// safe.
	// Just push one rectification with the zero DeploymentID.
	pair := sous.DeployablePair{}
	pair.SetID(sous.DeploymentID{Cluster: "cluster1", ManifestID: sous.ManifestID{
		Source: sous.SourceLocation{
			Repo: "github.com/opentable/repo1",
		},
	}})
	suite.componentLocator.QueueSet.Push(&sous.Rectification{
		Pair: pair,
	})
	if len(suite.componentLocator.QueueSet.Queues()) != 1 {
		panic("setup failed")
	}
	data := server.DeploymentQueuesResponse{}
	updater, err := suite.client.Retrieve("./all-deploy-queues", nil, &data, nil)
	suite.NoError(err)
	suite.Len(data.Queues, 1)
	suite.NotNil(updater)
}

func TestLiveServerSuite(t *testing.T) {
	suite.Run(t, new(liveServerSuite))
}

func TestInMemoryServerSuite(t *testing.T) {
	suite.Run(t, new(inmemServerSuite))
}
