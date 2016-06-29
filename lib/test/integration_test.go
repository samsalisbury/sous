package test

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

var imageName string

func TestMain(m *testing.M) {
	log.Print("hello there")
	flag.Parse()
	os.Exit(wrapCompose(m))
}

func TestGetLabels(t *testing.T) {
	registerLabelledContainers()
	assert := assert.New(t)
	cl := docker_registry.NewClient()
	cl.BecomeFoolishlyTrusting()

	labels, err := cl.LabelsForImageName(imageName)

	assert.Nil(err)
	assert.Contains(labels, sous.DockerRepoLabel)
	resetSingularity()
}

func TestGetRunningDeploymentSet(t *testing.T) {
	//	sous.Log.Debug.SetFlags(sous.Log.Debug.Flags() | log.Ltime)
	//	sous.Log.Debug.SetOutput(os.Stderr)
	//	sous.Log.Debug.Print("Starting stderr output")
	assert := assert.New(t)

	registerLabelledContainers()
	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()
	nc := sous.NewNameCache(drc, "sqlite3", sous.InMemoryConnection("grds"))
	ra := sous.NewRectiAgent(nc)

	deps, which := deploymentWithRepo(assert, ra, "https://github.com/opentable/docker-grafana.git")
	if assert.Equal(3, len(deps)) {
		grafana := deps[which]
		assert.Equal(singularityURL, grafana.Cluster)
		assert.Regexp("^0\\.1", grafana.Resources["cpus"])    // XXX strings and floats...
		assert.Regexp("^100\\.", grafana.Resources["memory"]) // XXX strings and floats...
		assert.Equal("1", grafana.Resources["ports"])         // XXX strings and floats...
		assert.Equal(17, grafana.SourceVersion.Version.Patch)
		assert.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceVersion.Version.Meta)
		assert.Equal(1, grafana.NumInstances)
		assert.Equal(sous.ManifestKindService, grafana.Kind)
	}

	resetSingularity()
}

func TestMissingImage(t *testing.T) {
	assert := assert.New(t)

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			singularityURL: sous.Cluster{
				BaseURL: singularityURL,
			},
		},
	}
	repoOne := "https://github.com/opentable/one.git"

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()
	// easiest way to make sure that the manifest doesn't actually get registered
	dummyNc := sous.NewNameCache(drc, "sqlite3", sous.InMemoryConnection("bitbucket"))

	stateOne := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"one": manifest(dummyNc, "opentable/one", "test-one", repoOne, "1.1.1"),
		},
	}

	// ****
	nc := sous.NewNameCache(drc, "sqlite3", sous.InMemoryConnection("missingimage"))
	ra := sous.NewRectiAgent(nc)
	err := sous.Resolve(ra, stateOne)
	assert.Error(err)

	// ****
	time.Sleep(1 * time.Second)

	_, which := deploymentWithRepo(assert, ra, repoOne)
	assert.Equal(which, -1, "opentable/one was deployed")

	resetSingularity()
}

func TestResolve(t *testing.T) {
	assert := assert.New(t)

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			singularityURL: sous.Cluster{
				BaseURL: singularityURL,
			},
		},
	}
	repoOne := "https://github.com/opentable/one.git"
	repoTwo := "https://github.com/opentable/two.git"
	repoThree := "https://github.com/opentable/three.git"

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()

	nc := sous.NewNameCache(drc, "sqlite3", sous.InMemoryConnection("testresolve"))
	ra := sous.NewRectiAgent(nc)

	stateOneTwo := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"one": manifest(nc, "opentable/one", "test-one", repoOne, "1.1.1"),
			"two": manifest(nc, "opentable/two", "test-two", repoTwo, "1.1.1"),
		},
	}
	stateTwoThree := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"two":   manifest(nc, "opentable/two", "test-two", repoTwo, "1.1.1"),
			"three": manifest(nc, "opentable/three", "test-three", repoThree, "1.1.1"),
		},
	}

	// ****
	log.Print("Resolving from nothing to one+two")
	err := sous.Resolve(ra, stateOneTwo)
	if err != nil {
		assert.Fail(err.Error())
	}
	// ****
	time.Sleep(3 * time.Second)

	deps, which := deploymentWithRepo(assert, ra, repoOne)
	if assert.NotEqual(which, -1, "opentable/one not successfully deployed") {
		one := deps[which]
		assert.Equal(1, one.NumInstances)
	}

	which = findRepo(deps, repoTwo)
	if assert.NotEqual(-1, which, "opentable/two not successfully deployed") {
		two := deps[which]
		assert.Equal(1, two.NumInstances)
	}

	// ****
	log.Println("Resolving from one+two to two+three")
	conflictRE := regexp.MustCompile(`Pending deploy already in progress`)

	// XXX Let's hope this is a temporary solution to a testing issue
	// The problem is laid out in DCOPS-7625
	for tries := 0; tries < 10; tries++ {
		err = sous.Resolve(ra, stateTwoThree)
		if err != nil {
			if !conflictRE.MatchString(err.Error()) {
				assert.Fail(err.Error())
			}
			log.Printf("Singularity conflict - waiting for previous deploy to complete - try #%d", tries+1)
			time.Sleep(1 * time.Second)
		}
	}

	if !assert.NoError(err) {
		assert.Fail(err)
	}
	// ****

	deps, which = deploymentWithRepo(assert, ra, repoTwo)
	if assert.NotEqual(-1, which, "opentable/two no longer deployed after resolve") {
		assert.Equal(1, deps[which].NumInstances)
	}

	which = findRepo(deps, repoThree)
	if assert.NotEqual(-1, which, "opentable/three not successfully deployed") {
		assert.Equal(1, deps[which].NumInstances)
		if assert.Len(deps[which].DeployConfig.Volumes, 1) {
			assert.Equal("RO", string(deps[which].DeployConfig.Volumes[0].Mode))
		}
	}

	which = findRepo(deps, repoOne)
	if which != -1 {
		assert.Equal(0, deps[which].NumInstances)
	}

	resetSingularity()
}

func deploymentWithRepo(assert *assert.Assertions, ra sous.RectificationClient, repo string) (sous.Deployments, int) {
	sc := sous.NewSetCollector(ra)
	deps, err := sc.GetRunningDeployment([]string{singularityURL})
	if assert.Nil(err) {
		return deps, findRepo(deps, repo)
	}
	return sous.Deployments{}, -1
}

func findRepo(deps sous.Deployments, repo string) int {
	for i := range deps {
		if deps[i] != nil {
			if deps[i].SourceVersion.RepoURL == sous.RepoURL(repo) {
				return i
			}
		}
	}
	return -1
}

func manifest(nc sous.ImageMapper, drepo, containerDir, sourceURL, version string) *sous.Manifest {
	sv := sous.SourceVersion{
		RepoURL:    sous.RepoURL(sourceURL),
		RepoOffset: sous.RepoOffset(""),
		Version:    semv.MustParse(version),
	}

	in := buildImageName(drepo, version)
	buildAndPushContainer(containerDir, in)

	nc.Insert(sv, in, "")

	return &sous.Manifest{
		Source: sous.SourceLocation{
			RepoURL:    sous.RepoURL(sourceURL),
			RepoOffset: sous.RepoOffset(""),
		},
		Owners: []string{`xyz`},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			singularityURL: sous.PartialDeploySpec{
				DeployConfig: sous.DeployConfig{
					Resources:    sous.Resources{}, //map[string]string
					Args:         []string{},
					Env:          sous.Env{}, //map[s]s
					NumInstances: 1,
					Volumes: sous.Volumes{
						&sous.Volume{"h", "c", sous.VolumeMode("RO")},
					},
				},
				Version: semv.MustParse(version),
				//clusterName: "it",
			},
		},
	}
}

func registerLabelledContainers() {
	registerAndDeploy(ip, "hello-labels", "hello-labels", []int32{})
	registerAndDeploy(ip, "hello-server-labels", "hello-server-labels", []int32{80})
	registerAndDeploy(ip, "grafana-repo", "grafana-labels", []int32{})
	imageName = fmt.Sprintf("%s/%s:%s", registryName, "grafana-repo", "latest")
}
