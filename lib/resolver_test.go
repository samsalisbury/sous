package sous

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuardImageMissing(t *testing.T) {
	assert := assert.New(t)

	svOne := MustParseSourceID(`github.com/ot/one,1.3.5`)
	dr := NewDummyRegistry()
	config := DeployConfig{NumInstances: 1}
	clusterX := &Cluster{Name: "x"}
	missing := Deployment{ClusterName: `x`, SourceID: svOne, DeployConfig: config, Cluster: clusterX}

	dr.FeedArtifact(nil, fmt.Errorf("dummy error"))
	_, err := guardImage(dr, &missing)
	assert.Error(err)
}

func TestGuardImageRejected(t *testing.T) {
	assert := assert.New(t)

	svTwo := MustParseSourceID(`github.com/ot/two,2.3.5`)
	dr := NewDummyRegistry()
	config := DeployConfig{NumInstances: 1}
	clusterX := &Cluster{Name: "x"}
	rejected := Deployment{ClusterName: `x`, SourceID: svTwo, DeployConfig: config, Cluster: clusterX}

	dr.FeedArtifact(&BuildArtifact{"ot-docker/one", "docker", []Quality{{"ephemeral_tag", "advisory"}}}, nil)

	_, err := guardImage(dr, &rejected)
	assert.Error(err)

}

func TestAllowUndeployedUglies(t *testing.T) {
	assert := assert.New(t)

	dr := NewDummyRegistry()
	svOne := MustParseSourceID(`github.com/ot/one,1.3.5`)
	config := DeployConfig{NumInstances: 0}
	borken := Deployment{ClusterName: `x`, SourceID: svOne, DeployConfig: config}

	dr.FeedArtifact(nil, fmt.Errorf("dummy error"))

	_, err := guardImage(dr, &borken)
	assert.NoError(err)
}

func TestAllowsWhitelistedAdvisories(t *testing.T) {
	assert := assert.New(t)

	svOne := MustParseSourceID(`github.com/ot/one,1.3.5`)
	dr := NewDummyRegistry()
	config := DeployConfig{NumInstances: 1}
	intoCI := Deployment{ClusterName: `ci`, Cluster: &Cluster{AllowedAdvisories: []string{"ephemeral_tag"}}, SourceID: svOne, DeployConfig: config}

	dr.FeedArtifact(&BuildArtifact{"ot-docker/one", "docker", []Quality{{"ephemeral_tag", "advisory"}}}, nil)

	art, err := guardImage(dr, &intoCI)
	assert.NoError(err)
	assert.NotNil(art)
}
