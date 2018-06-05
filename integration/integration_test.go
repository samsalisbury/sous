// +build integration

package integration

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/logging"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var imageName string

type integrationSuite struct {
	*assert.Assertions
	t         *testing.T
	req       *require.Assertions
	logbuf    *bytes.Buffer
	registry  docker_registry.Client
	nameCache *docker.NameCache
	client    *singularity.RectiAgent
	deployer  sous.Deployer
	ls        *logging.LogSet
	user      sous.User
}

func setupTest(t *testing.T) *integrationSuite {
	suite := &integrationSuite{
		t:          t,
		Assertions: assert.New(t),
		req:        require.New(t),
	}

	ResetSingularity()

	suite.logbuf = &bytes.Buffer{}
	suite.ls = logging.NewLogSet(semv.MustParse("0.0.0-integration"), "integration", "integration", suite.logbuf)
	suite.ls.BeChatty()

	suite.user = sous.User{}
	imageName = fmt.Sprintf("%s/%s:%s", registryName, "webapp", "latest")

	suite.registry = docker_registry.NewClient(suite.ls)
	suite.registry.BecomeFoolishlyTrusting()

	suite.T().Logf("New name cache for %q", t.Name())
	suite.nameCache = suite.newNameCache(suite.ls)
	suite.client = singularity.NewRectiAgent(suite.nameCache, suite.ls, suite.user)
	suite.deployer = singularity.NewDeployer(suite.client, suite.ls)
	return suite
}

func (suite *integrationSuite) T() *testing.T {
	return suite.t
}

func (suite *integrationSuite) Require() *require.Assertions {
	return suite.req
}

var none = sous.DeploymentID{}

func (suite *integrationSuite) deploymentWithRepo(clusterNames []string, repo string) (sous.DeployStates, sous.DeploymentID) {
	clusters := make(sous.Clusters, len(clusterNames))
	for _, name := range clusterNames {
		clusters[name] = &sous.Cluster{BaseURL: SingularityURL}
	}
	//suite.T().Logf("Calling RunningDeployments for clusters %#v", clusters)
	deps, err := suite.deployer.RunningDeployments(suite.nameCache, clusters)
	if suite.NoError(err) {
		return deps, suite.findRepo(deps, repo)
	}
	return sous.NewDeployStates(), none
}

func (suite *integrationSuite) findRepo(deps sous.DeployStates, repo string) sous.DeploymentID {
	for i, d := range deps.Snapshot() {
		if d != nil {
			if i.ManifestID.Source.Repo == repo {
				return i
			}
		}
	}
	return none
}

func (suite *integrationSuite) manifest(nc *docker.NameCache, drepo, containerDir, sourceURL, version string) *sous.Manifest {
	in := BuildImageName(drepo, version)
	if err := BuildAndPushContainer(suite.T(), containerDir, in); err != nil {
		suite.FailNow("setup failed to build and push container for %q: %s", in, err)
	}

	if nc != nil {
		_, err := nc.GetSourceID(docker.NewBuildArtifact(in, nil))
		if err != nil {
			suite.FailNow("setup failed to get source ID", err.Error())
		}
	}

	//checkReadyPath := "/health"

	//checkReadyTimeout := 500

	return &sous.Manifest{
		Source: sous.SourceLocation{
			Repo: sourceURL,
		},
		Owners: []string{`xyz`},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			"test-cluster": sous.DeploySpec{
				DeployConfig: sous.DeployConfig{
					Startup: sous.Startup{
						SkipCheck: true,
						//CheckReadyProtocol:   "HTTP",
						//CheckReadyURIPath:    checkReadyPath,
						//CheckReadyURITimeout: checkReadyTimeout,
						//Timeout:              checkReadyTimeout,
					},
					Resources:    sous.Resources{"cpus": "0.1", "memory": "100", "ports": "1"},
					Env:          sous.Env{"repo": drepo}, //map[s]s
					NumInstances: 1,
					Volumes:      sous.Volumes{{"/tmp", "/tmp", sous.VolumeMode("RO")}},
				},
				Version: semv.MustParse(version),
			},
		},
	}
}

func (suite *integrationSuite) newNameCache(ls logging.LogSink) *docker.NameCache {
	db := sous.SetupDB(suite.T())

	cache, err := docker.NewNameCache(registryName, suite.registry, ls, db)
	suite.Require().NoError(err)

	ids, err := cache.ListSourceIDs()
	suite.Require().NoError(err)
	suite.Require().Len(ids, 0, "Stale images in cache: %v", ids)

	return cache
}

func (suite *integrationSuite) waitUntilSettledStatus(clusters []string, sourceRepo string) *sous.DeployState {
	sleepTime := time.Duration(200) * time.Millisecond
	suite.T().Logf("Awaiting stabilization of Singularity deploy %q (either Active or Failed)...", sourceRepo)
	const waitLimit = 1500 // = 5 minutes max
	var deployState *sous.DeployState
	for counter := 1; counter < waitLimit; counter++ {
		ds, which := suite.deploymentWithRepo(clusters, sourceRepo)
		deps := ds.Snapshot()
		deployState = deps[which]
		suite.Require().NotNil(deployState, "deployState for %v (%q %q)", which, clusters, sourceRepo)
		if deployState.Status == sous.DeployStatusActive || deployState.Status == sous.DeployStatusFailed {
			suite.T().Logf("Stabilized with %s", deployState.Status)
			return deployState
		}
		time.Sleep(sleepTime)
	}
	suite.FailNow("Never stabilized", "%q didn't settle after %d polls; final status was %s", sourceRepo, waitLimit, deployState.Status)
	return nil
}

func (suite *integrationSuite) statusIs(ds *sous.DeployState, expected sous.DeployStatus) {
	actual := ds.Status
	suite.Equal(actual, expected, "deploy status is %q; want %q\n%s\nIn: %s", actual, expected, ds.ExecutorMessage, spew.Sdump(ds))
}

func (suite *integrationSuite) dumpLogs() {
	suite.T().Helper()
	suite.T().Log("Log buffer:\n" + suite.logbuf.String())
}

func (suite *integrationSuite) deployDefaultContainers() {
	suite.T().Log("Deploying default containers.")
	nilStartup := sous.Startup{SkipCheck: true}
	timeout := 500
	startup := sous.Startup{
		Timeout:            timeout,
		CheckReadyURIPath:  "/healthy",
		CheckReadyProtocol: "HTTP",
	}

	registerAndDeploy(suite.T(), "test-cluster", "hello-labels", "github.com/docker-library/hello-world", "hello-labels", "latest", []int32{}, nilStartup)
	registerAndDeploy(suite.T(), "test-cluster", "hello-server-labels", "github.com/docker/dockercloud-hello-world", "hello-server-labels", "latest", []int32{}, nilStartup)
	registerAndDeploy(suite.T(), "test-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)
	registerAndDeploy(suite.T(), "other-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)

	// This deployment fails immediately, and never results in a successful deployment at that singularity request.
	registerAndDeploy(suite.T(), "test-cluster", "supposed-to-fail", "github.com/opentable/homer-says-doh", "fails-labels", "1-fails", []int32{}, nilStartup)
	suite.T().Log("Deploying default containers; waiting for singularity.")
	WaitForSingularity()
}

func (suite *integrationSuite) withDeployment(clusters []string, repo string, count int, fn func(require.TestingT, *sous.DeployState)) {
	var deps map[sous.DeploymentID]*sous.DeployState
	for tries := 5; tries > 0; tries-- {
		ds, which := suite.deploymentWithRepo(clusters, repo)
		deps = ds.Snapshot()
		if len(deps) == count {
			fn(suite.T(), deps[which])
			return
		}
		time.Sleep(400 * time.Millisecond) // *5 is up to 2 seconds of sleep
	}
	suite.FailNow("Expected there to be %d deployments, but there are %d: \nDeployState map:\n%+#v", count, len(deps), deps)
}

func (suite *integrationSuite) tearDown() {
	sous.ReleaseDB(suite.T())

	if os.Getenv("INTEGRATION_LOGS") == "yes" {
		suite.dumpLogs()
	}
	// Previously, a ResetSingularity() was issued here. The ResetSingularity()
	// in the BeforeTest() already makes sure that Singularity is reset before
	// running a test case, so its presence here was redundant. With it gone, we
	// can look at the state of Singularity after a failed test.
}

// XXX I would like to move this to a separate file and tease out from it it's
// actual setup requirements (i.e. just the registry, not the whole external
// env.
func TestGetLabels(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()

	registerImage(t, "webapp", "webapp", "latest")
	suite.T().Logf("Getting labels for %s", imageName)
	labels, err := suite.registry.LabelsForImageName(imageName)

	suite.NoError(err)
	suite.Contains(labels, docker.DockerRepoLabel)
}

func TestNameCache(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()

	repoOne := "https://github.com/opentable/one.git"
	suite.manifest(suite.nameCache, "opentable/one", "test-one", repoOne, "1.1.1")

	cn, err := suite.nameCache.GetCanonicalName(BuildImageName("opentable/one", "1.1.1"))
	suite.Require().NoError(err)

	labels, err := suite.registry.LabelsForImageName(cn)
	suite.Require().NoError(err)

	suite.Equal("1.1.1", labels[docker.DockerVersionLabel])
}

func (suite *integrationSuite) depsCount(deps map[sous.DeploymentID]*sous.DeployState, count int) bool {
	if suite.Len(deps, count, "Expected there to be %d deployments, but there are %d: \nDeployState map:\n%+#v", count, len(deps), deps) {
		return true
	}
	return false
}

func TestGetRunningDeploymentSet_testCluster(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()

	suite.deployDefaultContainers()
	clusters := []string{"test-cluster"}

	// We run this test more than once to check that cache behaviour is
	// consistent whether the cache is already warmed up or not.
	const numberOfTestRuns = 2

	for i := 0; i < numberOfTestRuns; i++ {
		ds, which := suite.deploymentWithRepo(clusters, "github.com/example/webapp")
		deps := ds.Snapshot()
		if suite.depsCount(deps, 4) {
			webapp := deps[which]
			cacheHitText := fmt.Sprintf("on cache hit %d", i+1)
			suite.Equal(SingularityURL, webapp.Cluster.BaseURL, cacheHitText)
			suite.Regexp("^0\\.1", webapp.Resources["cpus"], cacheHitText)    // XXX strings and floats...
			suite.Regexp("^100\\.", webapp.Resources["memory"], cacheHitText) // XXX strings and floats...
			suite.Equal("1", webapp.Resources["ports"], cacheHitText)         // XXX strings and floats...
			suite.Equal(17, webapp.SourceID.Version.Patch, cacheHitText)
			//suite.Equal("91495f1b1630084e301241100ecf2e775f6b672c", webapp.SourceID.Version.Meta, cacheHitText) //991
			suite.Equal(1, webapp.NumInstances, cacheHitText)
			suite.Equal(sous.ManifestKindService, webapp.Kind, cacheHitText)
		} else {
			suite.T().Logf("Missing count occured in run #%d", i+1)
		}
	}
}

func TestGetRunningDeploymentSet_otherCluster(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()
	suite.deployDefaultContainers()
	clusters := []string{"other-cluster"}

	suite.withDeployment(clusters, "github.com/example/webapp", 1, func(_ require.TestingT, webapp *sous.DeployState) {
		suite.Equal(SingularityURL, webapp.Cluster.BaseURL)
		suite.Regexp("^0\\.1", webapp.Resources["cpus"])    // XXX strings and floats...
		suite.Regexp("^100\\.", webapp.Resources["memory"]) // XXX strings and floats...
		suite.Equal("1", webapp.Resources["ports"])         // XXX strings and floats...
		suite.Equal(17, webapp.SourceID.Version.Patch)
		//suite.Equal("91495f1b1630084e301241100ecf2e775f6b672c", webapp.SourceID.Version.Meta) //991
		suite.Equal(1, webapp.NumInstances)
		suite.Equal(sous.ManifestKindService, webapp.Kind)
	})

}

func TestGetRunningDeploymentSet_all(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()
	suite.deployDefaultContainers()
	clusters := []string{"test-cluster", "other-cluster"}

	suite.withDeployment(clusters, "github.com/example/webapp", 5, func(t require.TestingT, webapp *sous.DeployState) {
		assert.Equal(t, SingularityURL, webapp.Cluster.BaseURL)
		assert.Regexp(t, "^0\\.1", webapp.Resources["cpus"])    // XXX strings and floats...
		assert.Regexp(t, "^100\\.", webapp.Resources["memory"]) // XXX strings and floats...
		assert.Equal(t, "1", webapp.Resources["ports"])         // XXX strings and floats...
		assert.Equal(t, 17, webapp.SourceID.Version.Patch)
		//assert.Equal(t, "91495f1b1630084e301241100ecf2e775f6b672c", webapp.SourceID.Version.Meta) //991
		assert.Equal(t, 1, webapp.NumInstances)
		assert.Equal(t, sous.ManifestKindService, webapp.Kind)
	})
}

func TestFailedService(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()
	suite.deployDefaultContainers()
	clusters := []string{"test-cluster"}

	fails := suite.waitUntilSettledStatus(clusters, "github.com/opentable/homer-says-doh")
	suite.statusIs(fails, sous.DeployStatusFailed)
}

func TestFailedTimedOutService(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()
	timeout := 50
	uriPath := "slow-healthy"
	startup := sous.Startup{
		Timeout:              timeout,
		CheckReadyProtocol:   "HTTP",
		CheckReadyURIPath:    uriPath,
		CheckReadyURITimeout: timeout,
	}
	registerAndDeploy(t, "test-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)

	clusters := []string{"test-cluster"}
	fails := suite.waitUntilSettledStatus(clusters, "github.com/example/webapp")
	suite.statusIs(fails, sous.DeployStatusFailed)
}

func TestFailedNotHealthyService(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()
	timeout := 60
	uriPath := "sick"
	startup := sous.Startup{
		CheckReadyProtocol:   "HTTP",
		Timeout:              timeout,
		CheckReadyURIPath:    uriPath,
		CheckReadyURITimeout: timeout,
	}
	registerAndDeploy(t, "test-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)

	clusters := []string{"test-cluster"}
	fails := suite.waitUntilSettledStatus(clusters, "github.com/example/webapp")
	suite.statusIs(fails, sous.DeployStatusFailed)
}

func TestSuccessfulService(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()
	timeout := 300
	uriPath := "healthy"
	startup := sous.Startup{
		CheckReadyProtocol:   "HTTP",
		Timeout:              timeout,
		CheckReadyURIPath:    uriPath,
		CheckReadyURITimeout: timeout,
	}
	registerAndDeploy(t, "test-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)

	clusters := []string{"test-cluster"}

	succeeds := suite.waitUntilSettledStatus(clusters, "github.com/example/webapp")
	suite.statusIs(succeeds, sous.DeployStatusActive)
}

func TestFailedDeployFollowingSuccessfulDeploy(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()
	/* I am commenting out this block pursuant to the following note. Let's see how it does.
	/*
		If Travis passes after Fri Jul 21 10:52:27 PDT 2017 , remove this.
	if os.Getenv("CI") == "true" {
		// XXX means we need to do a desktop check before deploys
		suite.T().Skipf("On travis, we get 'Only 0 of 1 tasks could be launched for deploy, there may not be enough resources to launch the remaining tasks'")
	}
	*/
	clusters := []string{"test-cluster"}

	const sourceRepo = "github.com/user/succeedthenfail" // Part of request ID.
	const clusterName = "test-cluster"                   // Part of request ID.

	// Create an assert on a successful deployment.
	var ports []int32
	const repoName = "succeedthenfail"

	registerAndDeploy(t, clusterName, repoName, sourceRepo, "succeedthenfail-succeed", "1.0.0-succeed", ports, sous.Startup{
		SkipCheck: true,
	})

	deployState := suite.waitUntilSettledStatus(clusters, sourceRepo)
	suite.statusIs(deployState, sous.DeployStatusActive)

	if deployState.Status == sous.DeployStatusFailed {
		suite.T().Fatal("Aborting test, deploy failed.")
	}

	// Create an assert on a failed deployment.

	registerAndDeploy(t, clusterName, repoName, sourceRepo, "succeedthenfail-fail", "2.0.0-fail", ports, sous.Startup{SkipCheck: true})

	deployState = suite.waitUntilSettledStatus(clusters, sourceRepo)
	suite.statusIs(deployState, sous.DeployStatusFailed)
}

func TestMissingImage(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()
	suite.deployDefaultContainers()

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			"test-cluster": &sous.Cluster{
				BaseURL: SingularityURL,
			},
		},
	}
	repoOne := "github.com/opentable/one"

	stateOne := sous.State{
		Defs: clusterDefs,
		Manifests: sous.NewManifests(
			// easiest way to make sure that the manifest doesn't actually get registered
			suite.manifest(nil, "opentable/one", "test-one", repoOne, "1.1.1"),
		),
	}

	// ****
	rf := &sous.ResolveFilter{}
	sr := sous.NewDummyStateManager()
	sr.State = &stateOne
	qs := graph.NewR11nQueueSet(suite.deployer, suite.nameCache, rf, &graph.ServerStateManager{sr})
	r := sous.NewResolver(suite.deployer, suite.nameCache, rf, suite.ls, qs)

	deploymentsOne, err := stateOne.Deployments()
	suite.Require().NoError(err)

	err = r.Begin(deploymentsOne, clusterDefs.Clusters).Wait()

	suite.T().Logf("Missing Image Error: %v", err)
	suite.Error(err, "should report 'missing image' for opentable/one")

	// ****

	WaitForSingularity()

	clusters := []string{"test-cluster"}

	_, which := suite.deploymentWithRepo(clusters, repoOne)
	suite.Equal(which, none, "opentable/one was deployed, should not be")
}

func TestResolve(t *testing.T) {
	suite := setupTest(t)
	defer suite.tearDown()
	suite.deployDefaultContainers()
	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			"test-cluster": &sous.Cluster{
				BaseURL: SingularityURL,
			},
		},
	}
	repoOne := "github.com/opentable/one"
	repoTwo := "github.com/opentable/two"
	repoThree := "github.com/opentable/three"

	stateOneTwo := sous.State{
		Defs: clusterDefs,
		Manifests: sous.NewManifests(
			suite.manifest(suite.nameCache, "opentable/one", "test-one", repoOne, "1.1.1"),
			suite.manifest(suite.nameCache, "opentable/two", "test-two", repoTwo, "1.1.1"),
		),
	}
	deploymentsOneTwo, err := stateOneTwo.Deployments()
	suite.Require().NoError(err)

	stateTwoThree := sous.State{
		Defs: clusterDefs,
		Manifests: sous.NewManifests(
			suite.manifest(suite.nameCache, "opentable/two", "test-two-updated", repoTwo, "1.1.2"),
			suite.manifest(suite.nameCache, "opentable/three", "test-three", repoThree, "1.1.1"),
		),
	}
	deploymentsTwoThree, err := stateTwoThree.Deployments()
	suite.Require().NoError(err)

	logsink, logController := logging.NewLogSinkSpy()

	// ****
	rf := &sous.ResolveFilter{}
	sr := sous.NewDummyStateManager()
	sr.State = &stateOneTwo
	qs := graph.NewR11nQueueSet(suite.deployer, suite.nameCache, rf, &graph.ServerStateManager{sr})
	r := sous.NewResolver(suite.deployer, suite.nameCache, rf, logsink, qs)

	suite.T().Log("Begining OneTwo")
	err = r.Begin(deploymentsOneTwo, clusterDefs.Clusters).Wait()
	suite.T().Log("Finished OneTwo")
	if err != nil {
		suite.Fail(err.Error())
	}
	// ****
	//time.Sleep(3 * time.Second)
	WaitForSingularity()

	clusters := []string{"test-cluster"}
	ds, which := suite.deploymentWithRepo(clusters, repoOne)
	deps := ds.Snapshot()
	if suite.NotEqual(none, which, "opentable/one not successfully deployed") {
		one := deps[which]
		suite.Equal(1, one.NumInstances)
	}

	which = suite.findRepo(ds, repoTwo)
	if suite.NotEqual(none, which, "opentable/two not successfully deployed") {
		two := deps[which]
		suite.Equal(1, two.NumInstances)
	}

	dispositions := []string{}
	for _, call := range logController.CallsTo("Fields") {
		if lms, is := call.PassedArgs().Get(0).([]logging.EachFielder); is {
			for _, lm := range lms {
				lm.EachField(func(name logging.FieldName, val interface{}) {
					if disp, is := val.(string); is && name == logging.SousDiffDisposition {
						dispositions = append(dispositions, disp)
					}
				})
			}
		}
	}
	sort.Strings(dispositions)
	expectedDispositions := []string{"added", "added", "added", "added", "added", "added", "removed", "removed", "removed", "removed"}
	if !suite.Equal(expectedDispositions, dispositions) {
		log.Println(expectedDispositions, dispositions)
		log.Printf("All log messages:\n")
		for _, call := range logController.CallsTo("Fields") {
			if msgs, is := call.PassedArgs().Get(0).([]logging.EachFielder); is {
				m := map[logging.FieldName]interface{}{}
				for _, msg := range msgs {
					msg.EachField(func(k logging.FieldName, v interface{}) {
						m[k] = v
					})
				}
				log.Print(spew.Sprintf("%#v", m))
			} else {
				log.Printf("NOT A LOG MESSAGE: %+#v", call.PassedArgs().Get(1))
			}

		}

	}

	// ****
	log.Print("Resolving from one+two to two+three")

	// XXX Let's hope this is a temporary solution to a testing issue
	// The problem is laid out in DCOPS-7625
	for tries := 100; tries > 0; tries-- {
		client := singularity.NewRectiAgent(suite.nameCache, logsink, suite.user)
		deployer := singularity.NewDeployer(client, logging.SilentLogSet())

		rf := &sous.ResolveFilter{}
		sr := sous.NewDummyStateManager()
		sr.State = &stateOneTwo
		qs := graph.NewR11nQueueSet(suite.deployer, suite.nameCache, rf, &graph.ServerStateManager{sr})
		r := sous.NewResolver(deployer, suite.nameCache, rf, logging.SilentLogSet(), qs)

		err := r.Begin(deploymentsTwoThree, clusterDefs.Clusters).Wait()
		if !sous.AnyTransientResolveErrors(err) {
			break
		}

		//suite.Require().NotRegexp(`Pending deploy already in progress`, err.Error())
		suffix := `           this is dumb but it would suck to panic during tests
                                                                            what
																																						  is
																																							up
																																					gofmt?
			                                                                          `
		log.Printf("Singularity error:%s... - will try %d more times", spew.Sdump(err)[0:len(suffix)], tries)
		time.Sleep(2 * time.Second)
	}

	suite.Require().NoError(err)
	// ****

	ds, which = suite.deploymentWithRepo(clusters, repoTwo)
	deps = ds.Snapshot()
	if suite.NotEqual(none, which, "opentable/two no longer deployed after resolve") {
		dep := deps[which]
		suite.Equal(1, dep.NumInstances)
		suite.Equal("1.1.2", dep.Deployment.SourceID.Version.String())
	}

	which = suite.findRepo(ds, repoThree)
	if suite.NotEqual(none, which, "opentable/three not successfully deployed") {
		suite.Equal(1, deps[which].NumInstances)
		if suite.Len(deps[which].DeployConfig.Volumes, 1) {
			suite.Equal("RO", string(deps[which].DeployConfig.Volumes[0].Mode))
		}
	}

	// We no longer expect any deletions; See deployer.RectifySingleDelete.
	//expectedInstances := 0
	expectedInstances := 1

	which = suite.findRepo(ds, repoOne)
	if which != none {
		suite.Equal(expectedInstances, deps[which].NumInstances)
	}

}
