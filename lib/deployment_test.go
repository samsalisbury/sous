package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentEqual(t *testing.T) {
	assert := assert.New(t)

	dep := Deployment{}
	assert.True(dep.Equal(&Deployment{}))

	other := Deployment{
		Annotation: Annotation{
			RequestID: "somewhere around here",
		},
	}
	assert.True(dep.Equal(&other))
}

func TestCanonName(t *testing.T) {
	assert := assert.New(t)

	vers, _ := semv.Parse("1.2.3-test+thing")
	dep := Deployment{
		SourceVersion: SourceVersion{
			RepoURL:    RepoURL("one"),
			RepoOffset: RepoOffset("two"),
			Version:    vers,
		},
	}
	str := dep.SourceVersion.CanonicalName().String()
	assert.Regexp("one", str)
	assert.Regexp("two", str)
}

func TestBuildDeployment(t *testing.T) {
	assert := assert.New(t)
	m := &Manifest{
		Source: SourceLocation{},
		Owners: []string{"test@testerson.com"},
		Kind:   ManifestKindService,
	}
	sp := PartialDeploySpec{
		DeployConfig: DeployConfig{
			Resources:    Resources{},
			Args:         []string{},
			Env:          Env{},
			NumInstances: 3,
			Volumes: Volumes{
				&Volume{"h", "c", "RO"},
			},
		},
		Version:     semv.MustParse("1.2.3"),
		clusterName: "cluster.name",
	}
	ih := DeploymentSpecs{}

	d, err := BuildDeployment(m, sp, ih)

	if assert.NoError(err) {
		if assert.Len(d.DeployConfig.Volumes, 1) {
			assert.Equal("c", d.DeployConfig.Volumes[0].Container)
		}
	}
}
