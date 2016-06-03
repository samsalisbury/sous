package test

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

var imageName string

func TestMain(m *testing.M) {
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
	assert := assert.New(t)

	registerLabelledContainers()

	deps, which := deploymentWithRepo(assert, "https://github.com/opentable/docker-grafana.git")
	assert.Equal(3, len(deps))

	if which < 0 {
		assert.FailNow("If deployment is nil, other tests will crash")
	}

	grafana := deps[which]
	assert.Equal(singularityURL, grafana.Cluster)
	assert.Regexp("^0\\.1", grafana.Resources["cpus"])    // XXX strings and floats...
	assert.Regexp("^100\\.", grafana.Resources["memory"]) // XXX strings and floats...
	assert.Equal("1", grafana.Resources["ports"])         // XXX strings and floats...
	assert.Equal(17, grafana.SourceVersion.Version.Patch)
	assert.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceVersion.Version.Meta)
	assert.Equal(1, grafana.NumInstances)
	assert.Equal(sous.ManifestKindService, grafana.Kind)

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

	// easiest way to make sure that the manifest doesn't actually get registered
	dummyNc := sous.NewNameCache(docker_registry.NewClient(), "sqlite3", ":memory:")

	stateOne := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"one": manifest(dummyNc, "opentable/one", "test-one", repoOne, "1.1.1"),
		},
	}

	// ****
	nc := sous.NewNameCache(docker_registry.NewClient(), "sqlite3", ":memory:")
	err := sous.Resolve(nc, stateOne)
	assert.Error(err)

	// ****
	time.Sleep(1 * time.Second)

	_, which := deploymentWithRepo(assert, repoOne)
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

	nc := sous.NewNameCache(docker_registry.NewClient(), "sqlite3", ":memory:")

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
	err := sous.Resolve(nc, stateOneTwo)
	if err != nil {
		assert.Fail(err.Error())
	}
	// ****
	time.Sleep(1 * time.Second)

	deps, which := deploymentWithRepo(assert, repoOne)
	assert.NotEqual(which, -1, "opentable/one not successfully deployed")
	one := deps[which]
	assert.Equal(1, one.NumInstances)

	which = findRepo(deps, repoTwo)
	assert.NotEqual(-1, which, "opentable/two not successfully deployed")
	two := deps[which]
	assert.Equal(1, two.NumInstances)

	// ****
	err = sous.Resolve(nc, stateTwoThree)
	if err != nil {
		assert.Fail(err.Error())
	}
	// ****

	deps, which = deploymentWithRepo(assert, repoTwo)
	assert.NotEqual(-1, which, "opentable/two no longer deployed after resolve")
	assert.Equal(1, deps[which].NumInstances)

	which = findRepo(deps, repoThree)
	assert.NotEqual(-1, which, "opentable/three not successfully deployed")
	assert.Equal(1, deps[which].NumInstances)

	which = findRepo(deps, repoOne)
	if which != -1 {
		assert.Equal(0, deps[which].NumInstances)
	}

	resetSingularity()
}

func deploymentWithRepo(assert *assert.Assertions, repo string) (sous.Deployments, int) {
	deps, err := sous.GetRunningDeploymentSet([]string{singularityURL})
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

func manifest(nc sous.NameCache, drepo, containerDir, sourceURL, version string) *sous.Manifest {
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
