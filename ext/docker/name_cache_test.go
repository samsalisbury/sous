package docker

import (
	"flag"
	"os"
	"testing"

	"github.com/opentable/sous/integration"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(integration.WrapCompose(m, "../../test-registry"))
}

// TODO: copied from integration/integration_test.go, need to de-dupe
func manifest(nc sous.Registry, drepo, containerDir, sourceURL, version string) *sous.Manifest {
	//	sv := sous.SourceVersion{
	//		RepoURL:    sous.RepoURL(sourceURL),
	//		RepoOffset: sous.RepoOffset(""),
	//		Version:    semv.MustParse(version),
	//	}

	in := integration.BuildImageName(drepo, version)
	integration.BuildAndPushContainer(containerDir, in)

	//nc.Insert(sv, in, "")
	nc.GetSourceVersion(DockerBuildArtifact(in))

	return &sous.Manifest{
		Source: sous.SourceLocation{
			RepoURL:    sous.RepoURL(sourceURL),
			RepoOffset: sous.RepoOffset(""),
		},
		Owners: []string{`xyz`},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			integration.SingularityURL: sous.PartialDeploySpec{
				DeployConfig: sous.DeployConfig{
					Resources:    sous.Resources{"cpus": "0.1", "memory": "100", "ports": "1"}, //map[string]string
					Args:         []string{},
					Env:          sous.Env{"repo": drepo}, //map[s]s
					NumInstances: 1,
					Volumes:      sous.Volumes{&sous.Volume{"/tmp", "/tmp", sous.VolumeMode("RO")}},
				},
				Version: semv.MustParse(version),
				//clusterName: "it",
			},
		},
	}
}

func TestNameCache(t *testing.T) {
	assert := assert.New(t)
	sous.Log.Debug.SetOutput(os.Stdout)

	integration.ResetSingularity()
	defer integration.ResetSingularity()

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()

	db, err := GetDatabase(&DBConfig{"sqlite3", InMemoryConnection("testnamecache")})
	if err != nil {
		t.Fatal(err)
	}
	nc := NewNameCache(drc, db)

	repoOne := "https://github.com/opentable/one.git"
	manifest(nc, "opentable/one", "test-one", repoOne, "1.1.1")

	cn, err := nc.GetCanonicalName(integration.BuildImageName("opentable/one", "1.1.1"))
	if err != nil {
		assert.FailNow(err.Error())
	}
	labels, err := drc.LabelsForImageName(cn)

	if assert.NoError(err) {
		assert.Equal("1.1.1", labels[DockerVersionLabel])
	}
}
