package sous

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/samsalisbury/semv"
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
		SourceID: SourceID{
			SourceLocation: SourceLocation{
				Repo: "one",
				Dir:  "two",
			},
			Version: vers,
		},
	}
	str := dep.SourceID.SourceLocation.String()
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
	sp := DeploySpec{
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
	var ih []DeploySpec
	nick := "cn"

	state := &State{Defs: Defs{Clusters: Clusters{nick: &Cluster{BaseURL: "http://not"}}}}

	d, err := BuildDeployment(state, m, nick, sp, ih)

	if assert.NoError(err) {
		if assert.Len(d.DeployConfig.Volumes, 1) {
			assert.Equal("c", d.DeployConfig.Volumes[0].Container)
		}
		assert.Equal(nick, d.ClusterName)
	}
}
