package test

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/opentable/sous/sous"
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

	deps, err := sous.GetRunningDeploymentSet([]string{singularityURL})
	assert.Nil(err)
	assert.Equal(3, len(deps))
	var grafana *sous.Deployment
	for i := range deps {
		if deps[i].SourceVersion.RepoURL == "https://github.com/opentable/docker-grafana.git" {
			grafana = deps[i]
		}
	}
	if !assert.NotNil(grafana) {
		assert.FailNow("If deployment is nil, other tests will crash")
	}
	assert.Equal(singularityURL, grafana.Cluster)
	assert.Regexp("^0\\.1", grafana.Resources["cpus"])    // XXX strings and floats...
	assert.Regexp("^100\\.", grafana.Resources["memory"]) // XXX strings and floats...
	assert.Equal("1", grafana.Resources["ports"])         // XXX strings and floats...
	assert.Equal(17, grafana.SourceVersion.Patch)
	assert.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceVersion.Meta)
	assert.Equal(1, grafana.NumInstances)
	assert.Equal(sous.ManifestKindService, grafana.Kind)

	resetSingularity()
}

func TestResolve(t *testing.T) {
	assert := assert.New(t)

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			"it": sous.Cluster{
				BaseURL: singularityURL,
			},
		},
	}

	stateOneTwo := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"one": manifest("https://github.com/opentable/one", "1.1.1"),
			"two": manifest("https://github.com/opentable/two", "1.1.1"),
		},
	}
	stateTwoThree := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"two":   manifest("https://github.com/opentable/two", "1.1.1"),
			"three": manifest("https://github.com/opentable/three", "1.1.1"),
		},
	}

	Resolve(stateOneTwo)
	// one and two are running
	Resolve(stateTwoThree)
	// two and three are running, not one

	resetSingularity()
}

func manifest(sourceURL, version string) sous.Manifest {
	return sous.Manifest{
		Source: sous.SourceLocation{
			RepoURL:    sous.RepoURL(sourceURL),
			RepoOffset: sous.RepoOffset(""),
		},
		Owners: []string{`xyz`},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			"it": sous.PartialDeploySpec{
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
