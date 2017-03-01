// +build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/suite"
)

var imageName string

type integrationSuite struct {
	suite.Suite
	registry  docker_registry.Client
	nameCache *docker.NameCache
	client    *singularity.RectiAgent
	deployer  sous.Deployer
}

func TestIntegration(t *testing.T) {
	suite.Run(t, new(integrationSuite))
}

var none = sous.DeployID{}

func (suite *integrationSuite) deploymentWithRepo(clusterNames []string, repo string) (sous.DeployStates, sous.DeployID) {
	clusters := make(sous.Clusters, len(clusterNames))
	for _, name := range clusterNames {
		clusters[name] = &sous.Cluster{BaseURL: SingularityURL}
	}
	deps, err := suite.deployer.RunningDeployments(suite.nameCache, clusters)
	if suite.Nil(err) {
		return deps, suite.findRepo(deps, repo)
	}
	return sous.DeployStates{}, none
}

func (suite *integrationSuite) findRepo(deps sous.DeployStates, repo string) sous.DeployID {
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
	BuildAndPushContainer(containerDir, in)

	nc.GetSourceID(docker.NewBuildArtifact(in, nil))

	return &sous.Manifest{
		Source: sous.SourceLocation{
			Repo: sourceURL,
		},
		Owners: []string{`xyz`},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			"test-cluster": sous.DeploySpec{
				DeployConfig: sous.DeployConfig{
					Resources:    sous.Resources{"cpus": "0.1", "memory": "100", "ports": "1"},
					Args:         []string{},
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

	return docker.NewNameCache("", suite.registry, db)
}

func (suite *integrationSuite) BeforeTest(suiteName, testName string) {
	suite.T().Logf("Before %q %q", suiteName, testName)
	ResetSingularity()

	registerAndDeploy(ip, "test-cluster", "hello-labels", "github.com/docker-library/hello-world", "hello-labels", []int32{})
	registerAndDeploy(ip, "test-cluster", "hello-server-labels", "github.com/docker/dockercloud-hello-world", "hello-server-labels", []int32{8123})
	registerAndDeploy(ip, "test-cluster", "grafana-repo", "github.com/opentable/docker-grafana", "grafana-labels", []int32{})
	registerAndDeploy(ip, "other-cluster", "grafana-repo", "github.com/opentable/docker-grafana", "grafana-labels", []int32{})

	registerAndDeploy(ip, "test-cluster", "supposed-to-fail", "github.com/opentable/homer-says-doh", "fails-labels", []int32{})

	/*
		imageName := BuildImageName("github.com/opentable/homer-says-doh", "latest")
		err = BuildAndPushContainer("failed-labels", imageName)
		suite.Require().NoError(err)
	*/

	imageName = fmt.Sprintf("%s/%s:%s", registryName, "grafana-repo", "latest")

	suite.registry = docker_registry.NewClient()
	suite.registry.BecomeFoolishlyTrusting()

	suite.nameCache = suite.newNameCache(testName)
	suite.client = singularity.NewRectiAgent()
	suite.deployer = singularity.NewDeployer(suite.client)
}

func (suite *integrationSuite) TeardownTest() {
	//ResetSingularity()
}

func (suite *integrationSuite) TestGetLabels() {
	suite.T().Logf("%v %q", suite.registry, imageName)
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

func (suite *integrationSuite) TestGetRunningDeploymentSet_testCluster() {
	clusters := []string{"test-cluster"}

	// We run this test more than once to check that cache behaviour is
	// consistent whether the cache is already warmed up or not.
	const numberOfTestRuns = 2

	for i := 0; i < numberOfTestRuns; i++ {
		ds, which := suite.deploymentWithRepo(clusters, "github.com/opentable/docker-grafana")
		deps := ds.Snapshot()
		if suite.Equal(4, len(deps)) {
			grafana := deps[which]
			cacheHitText := fmt.Sprintf("on cache hit %d", i+1)
			suite.Equal(SingularityURL, grafana.Cluster.BaseURL, cacheHitText)
			suite.Regexp("^0\\.1", grafana.Resources["cpus"], cacheHitText)    // XXX strings and floats...
			suite.Regexp("^100\\.", grafana.Resources["memory"], cacheHitText) // XXX strings and floats...
			suite.Equal("1", grafana.Resources["ports"], cacheHitText)         // XXX strings and floats...
			suite.Equal(17, grafana.SourceID.Version.Patch, cacheHitText)
			suite.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceID.Version.Meta, cacheHitText)
			suite.Equal(1, grafana.NumInstances, cacheHitText)
			suite.Equal(sous.ManifestKindService, grafana.Kind, cacheHitText)
		}
	}

}

func (suite *integrationSuite) TestGetRunningDeploymentSet_otherCluster() {
	clusters := []string{"other-cluster"}

	ds, which := suite.deploymentWithRepo(clusters, "github.com/opentable/docker-grafana")
	deps := ds.Snapshot()
	if suite.Equal(1, len(deps)) {
		grafana := deps[which]
		suite.Equal(SingularityURL, grafana.Cluster.BaseURL)
		suite.Regexp("^0\\.1", grafana.Resources["cpus"])    // XXX strings and floats...
		suite.Regexp("^100\\.", grafana.Resources["memory"]) // XXX strings and floats...
		suite.Equal("1", grafana.Resources["ports"])         // XXX strings and floats...
		suite.Equal(17, grafana.SourceID.Version.Patch)
		suite.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceID.Version.Meta)
		suite.Equal(1, grafana.NumInstances)
		suite.Equal(sous.ManifestKindService, grafana.Kind)
	}

}

func (suite *integrationSuite) TestGetRunningDeploymentSet_all() {
	clusters := []string{"test-cluster", "other-cluster"}

	ds, which := suite.deploymentWithRepo(clusters, "github.com/opentable/docker-grafana")
	deps := ds.Snapshot()
	if suite.Equal(5, len(deps)) {
		grafana := deps[which]
		suite.Equal(SingularityURL, grafana.Cluster.BaseURL)
		suite.Regexp("^0\\.1", grafana.Resources["cpus"])    // XXX strings and floats...
		suite.Regexp("^100\\.", grafana.Resources["memory"]) // XXX strings and floats...
		suite.Equal("1", grafana.Resources["ports"])         // XXX strings and floats...
		suite.Equal(17, grafana.SourceID.Version.Patch)
		suite.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceID.Version.Meta)
		suite.Equal(1, grafana.NumInstances)
		suite.Equal(sous.ManifestKindService, grafana.Kind)
	}

}

func (suite *integrationSuite) TestFailedService() {
	clusters := []string{"test-cluster"}

	var fails *sous.DeployState
	for {
		ds, which := suite.deploymentWithRepo(clusters, "github.com/opentable/homer-says-doh")
		deps := ds.Snapshot()
		fails = deps[which]
		if fails == nil {
			suite.T().Log(which, len(deps), deps)
		}
		suite.Require().NotNil(fails)
		if fails.Status != sous.DeployStatusPending {
			break
		}
		suite.T().Log(which, fails, fails.Status)
		time.Sleep(time.Millisecond * 500)
	}
	suite.Equal(fails.Status, sous.DeployStatusFailed, "status was %s not %s", fails.Status, sous.DeployStatusFailed)
}

func (suite *integrationSuite) TestMissingImage() {
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
	r := sous.NewResolver(suite.deployer, suite.nameCache, &sous.ResolveFilter{})

	deploymentsOne, err := stateOne.Deployments()
	suite.Require().NoError(err)

	err = r.Begin(deploymentsOne, clusterDefs.Clusters).Wait()

	suite.Error(err)

	// ****
	time.Sleep(1 * time.Second)

	clusters := []string{"test-cluster"}

	_, which := suite.deploymentWithRepo(clusters, repoOne)
	suite.Equal(which, none, "opentable/one was deployed")
}

func (suite *integrationSuite) TestResolve() {
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
			suite.manifest(suite.nameCache, "opentable/two", "test-two", repoTwo, "1.1.1"),
			suite.manifest(suite.nameCache, "opentable/three", "test-three", repoThree, "1.1.1"),
		),
	}
	deploymentsTwoThree, err := stateTwoThree.Deployments()
	suite.Require().NoError(err)

	// ****
	r := sous.NewResolver(suite.deployer, suite.nameCache, &sous.ResolveFilter{})

	err = r.Begin(deploymentsOneTwo, clusterDefs.Clusters).Wait()
	if err != nil {
		suite.Fail(err.Error())
	}
	// ****
	time.Sleep(3 * time.Second)

	clusters := []string{"test-cluster"}
	ds, which := suite.deploymentWithRepo(clusters, repoOne)
	deps := ds.Snapshot()
	if suite.NotEqual(which, none, "opentable/one not successfully deployed") {
		one := deps[which]
		suite.Equal(1, one.NumInstances)
	}

	which = suite.findRepo(ds, repoTwo)
	if suite.NotEqual(none, which, "opentable/two not successfully deployed") {
		two := deps[which]
		suite.Equal(1, two.NumInstances)
	}

	// ****
	suite.T().Log("Resolving from one+two to two+three")

	// XXX Let's hope this is a temporary solution to a testing issue
	// The problem is laid out in DCOPS-7625
	for tries := 0; tries < 3; tries++ {
		client := singularity.NewRectiAgent()
		deployer := singularity.NewDeployer(client)

		r := sous.NewResolver(deployer, suite.nameCache, &sous.ResolveFilter{})

		err := r.Begin(deploymentsTwoThree, clusterDefs.Clusters).Wait()
		if err != nil {
			suite.Require().NotRegexp(`Pending deploy already in progress`, err.Error())

			suite.T().Logf("Singularity conflict - waiting for previous deploy to complete - try #%d", tries+1)
			time.Sleep(1 * time.Second)
		}
	}

	suite.Require().NoError(err)
	// ****

	ds, which = suite.deploymentWithRepo(clusters, repoTwo)
	deps = ds.Snapshot()
	if suite.NotEqual(none, which, "opentable/two no longer deployed after resolve") {
		suite.Equal(1, deps[which].NumInstances)
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
