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
	"github.com/stretchr/testify/suite"
)

var imageName string

type integrationSuite struct {
	suite.Suite
	logbuf    *bytes.Buffer
	registry  docker_registry.Client
	nameCache *docker.NameCache
	client    *singularity.RectiAgent
	deployer  sous.Deployer
}

func TestIntegration(t *testing.T) {
	suite.Run(t, new(integrationSuite))
}

var none = sous.DeploymentID{}

func (suite *integrationSuite) deploymentWithRepo(clusterNames []string, repo string) (sous.DeployStates, sous.DeploymentID) {
	clusters := make(sous.Clusters, len(clusterNames))
	for _, name := range clusterNames {
		clusters[name] = &sous.Cluster{BaseURL: SingularityURL}
	}
	suite.T().Logf("Calling RunningDeployments for clusters %#v", clusters)
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
	if err := BuildAndPushContainer(containerDir, in); err != nil {
		suite.FailNow("setup failed to build and push container for %q: %s", in, err)
	}

	_, err := nc.GetSourceID(docker.NewBuildArtifact(in, nil))
	if err != nil {
		suite.FailNow("setup failed to get source ID: %s", err)
	}

	checkReadyPath := "/health"

	checkReadyTimeout := 500

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
						CheckReadyProtocol:   "HTTP",
						CheckReadyURIPath:    checkReadyPath,
						CheckReadyURITimeout: checkReadyTimeout,
						Timeout:              checkReadyTimeout,
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

func (suite *integrationSuite) newNameCache(name string) *docker.NameCache {
	db, err := docker.GetDatabase(&docker.DBConfig{
		Driver:     "sqlite3_sous",
		Connection: docker.InMemoryConnection(name),
	})

	suite.Require().NoError(err)

	cache, err := docker.NewNameCache(registryName, suite.registry, logging.SilentLogSet(), db)
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
		suite.Require().NotNil(deployState)
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

func (suite *integrationSuite) BeforeTest(suiteName, testName string) {
	ResetSingularity()

	suite.logbuf = &bytes.Buffer{}
	logset := logging.NewLogSet(semv.MustParse("0.0.0-integration"), "integration", "integration", suite.logbuf)
	logset.BeChatty()

	imageName = fmt.Sprintf("%s/%s:%s", registryName, "webapp", "latest")

	suite.registry = docker_registry.NewClient(logset)
	suite.registry.BecomeFoolishlyTrusting()

	suite.T().Logf("New name cache for %q", testName)
	suite.nameCache = suite.newNameCache(testName)
	suite.client = singularity.NewRectiAgent(suite.nameCache)
	suite.deployer = singularity.NewDeployer(suite.client, logset)
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

	registerAndDeploy(ip, "test-cluster", "hello-labels", "github.com/docker-library/hello-world", "hello-labels", "latest", []int32{}, nilStartup)
	registerAndDeploy(ip, "test-cluster", "hello-server-labels", "github.com/docker/dockercloud-hello-world", "hello-server-labels", "latest", []int32{}, nilStartup)
	registerAndDeploy(ip, "test-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)
	registerAndDeploy(ip, "other-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)

	// This deployment fails immediately, and never results in a successful deployment at that singularity request.
	registerAndDeploy(ip, "test-cluster", "supposed-to-fail", "github.com/opentable/homer-says-doh", "fails-labels", "1-fails", []int32{}, nilStartup)
	suite.T().Log("Deploying default containers; waiting for singularity.")
	WaitForSingularity()
}

func (suite *integrationSuite) TearDownTest() {
	if os.Getenv("INTEGRATION_LOGS") == "yes" {
		suite.dumpLogs()
	}
	// Previously, a ResetSingularity() was issued here. The ResetSingularity()
	// in the BeforeTest() already makes sure that Singularity is reset before
	// running a test case, so its presence here was redundant. With it gone, we
	// can look at the state of Singularity after a failed test.
}

func (suite *integrationSuite) TestGetLabels() {
	labels, err := suite.registry.LabelsForImageName(imageName)

	suite.Nil(err)
	suite.Contains(labels, docker.DockerRepoLabel)
}

func (suite *integrationSuite) TestNameCache() {
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
	suite.dumpLogs()
	return false
}

func (suite *integrationSuite) TestGetRunningDeploymentSet_testCluster() {
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
			suite.Equal("91495f1b1630084e301241100ecf2e775f6b672c", webapp.SourceID.Version.Meta, cacheHitText)
			suite.Equal(1, webapp.NumInstances, cacheHitText)
			suite.Equal(sous.ManifestKindService, webapp.Kind, cacheHitText)
		} else {
			suite.T().Logf("Missing count occured in run #%d", i+1)
		}
	}
}

func (suite *integrationSuite) TestGetRunningDeploymentSet_otherCluster() {
	suite.deployDefaultContainers()
	clusters := []string{"other-cluster"}

	ds, which := suite.deploymentWithRepo(clusters, "github.com/example/webapp")
	deps := ds.Snapshot()
	if suite.depsCount(deps, 1) {
		webapp := deps[which]
		suite.Equal(SingularityURL, webapp.Cluster.BaseURL)
		suite.Regexp("^0\\.1", webapp.Resources["cpus"])    // XXX strings and floats...
		suite.Regexp("^100\\.", webapp.Resources["memory"]) // XXX strings and floats...
		suite.Equal("1", webapp.Resources["ports"])         // XXX strings and floats...
		suite.Equal(17, webapp.SourceID.Version.Patch)
		suite.Equal("91495f1b1630084e301241100ecf2e775f6b672c", webapp.SourceID.Version.Meta)
		suite.Equal(1, webapp.NumInstances)
		suite.Equal(sous.ManifestKindService, webapp.Kind)
	}

}

func (suite *integrationSuite) TestGetRunningDeploymentSet_all() {
	suite.deployDefaultContainers()
	clusters := []string{"test-cluster", "other-cluster"}

	ds, which := suite.deploymentWithRepo(clusters, "github.com/example/webapp")
	deps := ds.Snapshot()
	if suite.depsCount(deps, 5) {
		webapp := deps[which]
		suite.Equal(SingularityURL, webapp.Cluster.BaseURL)
		suite.Regexp("^0\\.1", webapp.Resources["cpus"])    // XXX strings and floats...
		suite.Regexp("^100\\.", webapp.Resources["memory"]) // XXX strings and floats...
		suite.Equal("1", webapp.Resources["ports"])         // XXX strings and floats...
		suite.Equal(17, webapp.SourceID.Version.Patch)
		suite.Equal("91495f1b1630084e301241100ecf2e775f6b672c", webapp.SourceID.Version.Meta)
		suite.Equal(1, webapp.NumInstances)
		suite.Equal(sous.ManifestKindService, webapp.Kind)
	}

}

func (suite *integrationSuite) TestFailedService() {
	suite.deployDefaultContainers()
	clusters := []string{"test-cluster"}

	fails := suite.waitUntilSettledStatus(clusters, "github.com/opentable/homer-says-doh")
	suite.statusIs(fails, sous.DeployStatusFailed)
}

func (suite *integrationSuite) TestFailedTimedOutService() {
	timeout := 50
	uriPath := "slow-healthy"
	startup := sous.Startup{
		Timeout:              timeout,
		CheckReadyProtocol:   "HTTP",
		CheckReadyURIPath:    uriPath,
		CheckReadyURITimeout: timeout,
	}
	registerAndDeploy(ip, "test-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)

	clusters := []string{"test-cluster"}
	fails := suite.waitUntilSettledStatus(clusters, "github.com/example/webapp")
	suite.statusIs(fails, sous.DeployStatusFailed)
}

func (suite *integrationSuite) TestFailedNotHealthyService() {
	timeout := 60
	uriPath := "sick"
	startup := sous.Startup{
		CheckReadyProtocol:   "HTTP",
		Timeout:              timeout,
		CheckReadyURIPath:    uriPath,
		CheckReadyURITimeout: timeout,
	}
	registerAndDeploy(ip, "test-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)

	clusters := []string{"test-cluster"}
	fails := suite.waitUntilSettledStatus(clusters, "github.com/example/webapp")
	suite.statusIs(fails, sous.DeployStatusFailed)
}

func (suite *integrationSuite) TestSuccessfulService() {
	timeout := 300
	uriPath := "healthy"
	startup := sous.Startup{
		CheckReadyProtocol:   "HTTP",
		Timeout:              timeout,
		CheckReadyURIPath:    uriPath,
		CheckReadyURITimeout: timeout,
	}
	registerAndDeploy(ip, "test-cluster", "webapp", "github.com/example/webapp", "webapp", "latest", []int32{}, startup)

	clusters := []string{"test-cluster"}

	succeeds := suite.waitUntilSettledStatus(clusters, "github.com/example/webapp")
	suite.statusIs(succeeds, sous.DeployStatusActive)
}

func (suite *integrationSuite) TestFailedDeployFollowingSuccessfulDeploy() {
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

	registerAndDeploy(ip, clusterName, repoName, sourceRepo, "succeedthenfail-succeed", "1.0.0-succeed", ports, sous.Startup{
		SkipCheck: true,
	})

	deployState := suite.waitUntilSettledStatus(clusters, sourceRepo)
	suite.statusIs(deployState, sous.DeployStatusActive)

	if deployState.Status == sous.DeployStatusFailed {
		suite.T().Fatal("Aborting test, deploy failed.")
	}

	// Create an assert on a failed deployment.

	registerAndDeploy(ip, clusterName, repoName, sourceRepo, "succeedthenfail-fail", "2.0.0-fail", ports, sous.Startup{SkipCheck: true})

	deployState = suite.waitUntilSettledStatus(clusters, sourceRepo)
	suite.statusIs(deployState, sous.DeployStatusFailed)
}

func (suite *integrationSuite) TestMissingImage() {
	suite.deployDefaultContainers()

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			"test-cluster": &sous.Cluster{
				BaseURL: SingularityURL,
			},
		},
	}
	repoOne := "github.com/opentable/one"

	// easiest way to make sure that the manifest doesn't actually get registered
	dummyNc := suite.newNameCache("devnull")

	stateOne := sous.State{
		Defs: clusterDefs,
		Manifests: sous.NewManifests(
			suite.manifest(dummyNc, "opentable/one", "test-one", repoOne, "1.1.1"),
		),
	}

	// ****
	rf := &sous.ResolveFilter{}
	sr := sous.NewDummyStateManager()
	sr.State = &stateOne
	qs := graph.NewR11nQueueSet(suite.deployer, suite.nameCache, rf, &graph.ServerStateManager{sr})
	r := sous.NewResolver(suite.deployer, suite.nameCache, rf, logging.SilentLogSet(), qs)

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

func (suite *integrationSuite) TestResolve() {
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
	for _, call := range logController.CallsTo("LogMessage") {
		if lm, is := call.PassedArgs().Get(1).(logging.LogMessage); is {
			lm.EachField(func(name string, val interface{}) {
				if disp, is := val.(string); is && name == "sous-diff-disposition" {
					dispositions = append(dispositions, disp)
				}
			})
		}
	}
	sort.Strings(dispositions)
	expectedDispositions := []string{"added", "added", "removed", "removed", "removed", "removed"}
	if !suite.Equal(expectedDispositions, dispositions) {
		log.Printf("All log messages:\n")
		for _, call := range logController.CallsTo("LogMessage") {
			if msg, is := call.PassedArgs().Get(1).(logging.LogMessage); is {
				m := map[string]interface{}{}
				msg.EachField(func(k string, v interface{}) {
					m[k] = v
				})
				m["message"] = msg.Message()
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
		client := singularity.NewRectiAgent(suite.nameCache)
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
