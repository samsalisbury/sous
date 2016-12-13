// +build integration

package integration

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
)

var imageName string

func TestGetLabels(t *testing.T) {
	registerLabelledContainers()
	assert := assert.New(t)
	cl := docker_registry.NewClient()
	cl.BecomeFoolishlyTrusting()

	labels, err := cl.LabelsForImageName(imageName)

	assert.Nil(err)
	assert.Contains(labels, docker.DockerRepoLabel)
	ResetSingularity()
}

func newInMemoryDB(name string) *sql.DB {
	db, err := docker.GetDatabase(&docker.DBConfig{
		Driver:     "sqlite3_sous",
		Connection: docker.InMemoryConnection(name),
	})
	if err != nil {
		panic(err)
	}
	return db
}

func TestGetRunningDeploymentSet_testCluster(t *testing.T) {
	//sous.Log.Vomit.SetFlags(sous.Log.Vomit.Flags() | log.Ltime)
	//sous.Log.Vomit.SetOutput(os.Stderr)
	//sous.Log.Vomit.Print("Starting stderr output")
	sous.Log.Debug.SetFlags(sous.Log.Debug.Flags() | log.Ltime)
	sous.Log.Debug.SetOutput(os.Stderr)
	sous.Log.Debug.Print("Starting stderr output")
	assert := assert.New(t)

	registerLabelledContainers()
	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()
	nc := docker.NewNameCache("", drc, newInMemoryDB("grds"))
	client := singularity.NewRectiAgent()
	d := singularity.NewDeployer(client)

	clusters := []string{"test-cluster"}

	ds, which := deploymentWithRepo(clusters, nc, assert, d, "github.com/opentable/docker-grafana")
	deps := ds.Snapshot()
	if assert.Equal(3, len(deps)) {
		grafana := deps[which]
		assert.Equal(SingularityURL, grafana.Cluster.BaseURL)
		assert.Regexp("^0\\.1", grafana.Resources["cpus"])    // XXX strings and floats...
		assert.Regexp("^100\\.", grafana.Resources["memory"]) // XXX strings and floats...
		assert.Equal("1", grafana.Resources["ports"])         // XXX strings and floats...
		assert.Equal(17, grafana.SourceID.Version.Patch)
		assert.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceID.Version.Meta)
		assert.Equal(1, grafana.NumInstances)
		assert.Equal(sous.ManifestKindService, grafana.Kind)
	}

	ResetSingularity()
}

func TestGetRunningDeploymentSet_testCluster2(t *testing.T) {
	//sous.Log.Vomit.SetFlags(sous.Log.Vomit.Flags() | log.Ltime)
	//sous.Log.Vomit.SetOutput(os.Stderr)
	//sous.Log.Vomit.Print("Starting stderr output")
	sous.Log.Debug.SetFlags(sous.Log.Debug.Flags() | log.Ltime)
	sous.Log.Debug.SetOutput(os.Stderr)
	sous.Log.Debug.Print("Starting stderr output")
	assert := assert.New(t)

	registerLabelledContainers()
	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()
	nc := docker.NewNameCache("", drc, newInMemoryDB("grds"))
	client := singularity.NewRectiAgent()
	d := singularity.NewDeployer(client)

	clusters := []string{"test-cluster"}

	ds, which := deploymentWithRepo(clusters, nc, assert, d, "github.com/opentable/docker-grafana")
	deps := ds.Snapshot()
	if assert.Equal(3, len(deps)) {
		grafana := deps[which]
		assert.Equal(SingularityURL, grafana.Cluster.BaseURL)
		assert.Regexp("^0\\.1", grafana.Resources["cpus"])    // XXX strings and floats...
		assert.Regexp("^100\\.", grafana.Resources["memory"]) // XXX strings and floats...
		assert.Equal("1", grafana.Resources["ports"])         // XXX strings and floats...
		assert.Equal(17, grafana.SourceID.Version.Patch)
		assert.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceID.Version.Meta)
		assert.Equal(1, grafana.NumInstances)
		assert.Equal(sous.ManifestKindService, grafana.Kind)
	}

	ResetSingularity()
}

func TestMissingImage(t *testing.T) {
	assert := assert.New(t)

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			"test-cluster": &sous.Cluster{
				BaseURL: SingularityURL,
			},
		},
	}
	repoOne := "github.com/opentable/one"

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()
	// easiest way to make sure that the manifest doesn't actually get registered
	dummyNc := docker.NewNameCache("", drc, newInMemoryDB("bitbucket"))

	stateOne := sous.State{
		Defs: clusterDefs,
		Manifests: sous.NewManifests(
			manifest(dummyNc, "opentable/one", "test-one", repoOne, "1.1.1"),
		),
	}

	// ****
	nc := docker.NewNameCache("", drc, newInMemoryDB("missingimage"))

	client := singularity.NewRectiAgent()
	deployer := singularity.NewDeployer(client)

	r := sous.NewResolver(deployer, nc, &sous.ResolveFilter{})

	deploymentsOne, err := stateOne.Deployments()
	if err != nil {
		t.Fatal(err)
	}
	err = r.Resolve(deploymentsOne, clusterDefs.Clusters)

	assert.Error(err)

	// ****
	time.Sleep(1 * time.Second)

	clusters := []string{"test-cluster"}

	_, which := deploymentWithRepo(clusters, nc, assert, deployer, repoOne)
	assert.Equal(which, none, "opentable/one was deployed")

	ResetSingularity()
}

func TestResolve(t *testing.T) {
	assert := assert.New(t)
	//sous.Log.Vomit.SetOutput(os.Stderr)
	sous.Log.Debug.SetOutput(os.Stderr)

	ResetSingularity()
	defer ResetSingularity()

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

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()

	db := newInMemoryDB("testresolve")

	nc := docker.NewNameCache("", drc, db)

	stateOneTwo := sous.State{
		Defs: clusterDefs,
		Manifests: sous.NewManifests(
			manifest(nc, "opentable/one", "test-one", repoOne, "1.1.1"),
			manifest(nc, "opentable/two", "test-two", repoTwo, "1.1.1"),
		),
	}
	deploymentsOneTwo, err := stateOneTwo.Deployments()
	if err != nil {
		t.Fatal(err)
	}
	stateTwoThree := sous.State{
		Defs: clusterDefs,
		Manifests: sous.NewManifests(
			manifest(nc, "opentable/two", "test-two", repoTwo, "1.1.1"),
			manifest(nc, "opentable/three", "test-three", repoThree, "1.1.1"),
		),
	}
	deploymentsTwoThree, err := stateTwoThree.Deployments()
	if err != nil {
		t.Fatal(err)
	}

	// ****
	log.Print("Resolving from nothing to one+two")
	client := singularity.NewRectiAgent()
	deployer := singularity.NewDeployer(client)

	r := sous.NewResolver(deployer, nc, &sous.ResolveFilter{})

	err = r.Resolve(deploymentsOneTwo, clusterDefs.Clusters)
	if err != nil {
		assert.Fail(err.Error())
	}
	// ****
	time.Sleep(3 * time.Second)

	clusters := []string{"test-cluster"}
	ds, which := deploymentWithRepo(clusters, nc, assert, deployer, repoOne)
	deps := ds.Snapshot()
	if assert.NotEqual(which, none, "opentable/one not successfully deployed") {
		one := deps[which]
		assert.Equal(1, one.NumInstances)
	}

	which = findRepo(ds, repoTwo)
	if assert.NotEqual(none, which, "opentable/two not successfully deployed") {
		two := deps[which]
		assert.Equal(1, two.NumInstances)
	}

	// ****
	log.Println("Resolving from one+two to two+three")
	conflictRE := regexp.MustCompile(`Pending deploy already in progress`)

	// XXX Let's hope this is a temporary solution to a testing issue
	// The problem is laid out in DCOPS-7625
	for tries := 0; tries < 3; tries++ {
		client := singularity.NewRectiAgent()
		deployer := singularity.NewDeployer(client)

		r := sous.NewResolver(deployer, nc, &sous.ResolveFilter{})

		err := r.Resolve(deploymentsTwoThree, clusterDefs.Clusters)
		if err != nil {
			if !conflictRE.MatchString(err.Error()) {
				assert.FailNow(err.Error())
			}
			log.Printf("Singularity conflict - waiting for previous deploy to complete - try #%d", tries+1)
			time.Sleep(1 * time.Second)
		}
	}

	if !assert.NoError(err) {
		assert.Fail(err.Error())
	}
	// ****

	ds, which = deploymentWithRepo(clusters, nc, assert, deployer, repoTwo)
	deps = ds.Snapshot()
	if assert.NotEqual(none, which, "opentable/two no longer deployed after resolve") {
		assert.Equal(1, deps[which].NumInstances)
	}

	which = findRepo(ds, repoThree)
	if assert.NotEqual(none, which, "opentable/three not successfully deployed") {
		assert.Equal(1, deps[which].NumInstances)
		if assert.Len(deps[which].DeployConfig.Volumes, 1) {
			assert.Equal("RO", string(deps[which].DeployConfig.Volumes[0].Mode))
		}
	}

	// We no longer expect any deletions; See deployer.RectifySingleDelete.
	//expectedInstances := 0
	expectedInstances := 1

	which = findRepo(ds, repoOne)
	if which != none {
		assert.Equal(expectedInstances, deps[which].NumInstances)
	}

}

var none = sous.DeployID{}

func deploymentWithRepo(clusterNames []string, reg sous.Registry, assert *assert.Assertions, sc sous.Deployer, repo string) (sous.Deployments, sous.DeployID) {
	clusters := make(sous.Clusters, len(clusterNames))
	for _, name := range clusterNames {
		clusters[name] = &sous.Cluster{BaseURL: SingularityURL}
	}
	deps, err := sc.RunningDeployments(reg, clusters)
	if assert.Nil(err) {
		return deps, findRepo(deps, repo)
	}
	return sous.Deployments{}, none
}

func findRepo(deps sous.Deployments, repo string) sous.DeployID {
	for i, d := range deps.Snapshot() {
		if d != nil {
			if i.ManifestID.Source.Repo == repo {
				return i
			}
		}
	}
	return none
}

func manifest(nc sous.Registry, drepo, containerDir, sourceURL, version string) *sous.Manifest {
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

func registerLabelledContainers() {
	registerAndDeploy(ip, "test-cluster", "hello-labels", "hello-labels", []int32{})
	registerAndDeploy(ip, "test-cluster", "hello-server-labels", "hello-server-labels", []int32{80})
	registerAndDeploy(ip, "test-cluster", "grafana-repo", "grafana-labels", []int32{})
	imageName = fmt.Sprintf("%s/%s:%s", registryName, "grafana-repo", "latest")
}
