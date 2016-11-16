package sous

import (
	"fmt"
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/nyarly/testify/require"
	"github.com/pkg/errors"
)

func TestGuardImages(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	svOne := MustParseSourceID(`github.com/ot/one,1.3.5`)
	svTwo := MustParseSourceID(`github.com/ot/two,2.3.5`)
	dr := NewDummyRegistry()
	config := DeployConfig{NumInstances: 1}
	clusterX := &Cluster{Name: "x"}
	missing := Deployment{ClusterName: `x`, SourceID: svOne, DeployConfig: config, Cluster: clusterX}
	rejected := Deployment{ClusterName: `x`, SourceID: svTwo, DeployConfig: config, Cluster: clusterX}
	gdm := MakeDeployments(2)
	gdm.Add(&missing)
	gdm.Add(&rejected)

	dr.FeedArtifact(nil, fmt.Errorf("dummy error"))
	dr.FeedArtifact(&BuildArtifact{"ot-docker/one", "docker", []Quality{{"ephemeral_tag", "advisory"}}}, nil)

	err := GuardImages(dr, gdm)
	if err == nil {
		t.Fatalf("got nil; want an error")
	}

	resolveErrors, ok := errors.Cause(err).(*ResolveErrors)
	if !ok {
		t.Fatalf("got error type %T (%q); want a *ResolveErrors", err, err)
	}
	assert.Error(resolveErrors)
	require.Len(resolveErrors.Causes, 2)
}

func TestAllowUndeployedUglies(t *testing.T) {
	assert := assert.New(t)

	dr := NewDummyRegistry()
	svOne := MustParseSourceID(`github.com/ot/one,1.3.5`)
	config := DeployConfig{NumInstances: 0}
	borken := Deployment{ClusterName: `x`, SourceID: svOne, DeployConfig: config}
	gdm := MakeDeployments(1)
	gdm.Add(&borken)

	dr.FeedArtifact(nil, fmt.Errorf("dummy error"))

	assert.NoError(GuardImages(dr, gdm))
}

func TestAllowsWhitelistedAdvisories(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	svOne := MustParseSourceID(`github.com/ot/one,1.3.5`)
	dr := NewDummyRegistry()
	config := DeployConfig{NumInstances: 1}
	intoCI := Deployment{ClusterName: `ci`, Cluster: &Cluster{AllowedAdvisories: []string{"ephemeral_tag"}}, SourceID: svOne, DeployConfig: config}
	intoProd := Deployment{ClusterName: `prod`, Cluster: &Cluster{}, SourceID: svOne, DeployConfig: config}
	gdm := MakeDeployments(2)
	gdm.Add(&intoCI)
	gdm.Add(&intoProd)

	dr.FeedArtifact(&BuildArtifact{"ot-docker/one", "docker", []Quality{{"ephemeral_tag", "advisory"}}}, nil)
	dr.FeedArtifact(&BuildArtifact{"ot-docker/one", "docker", []Quality{{"ephemeral_tag", "advisory"}}}, nil)

	err, ok := errors.Cause(GuardImages(dr, gdm)).(*ResolveErrors)
	require.True(ok)
	assert.Error(err)
	require.Len(err.Causes, 1)
	require.IsType(&UnacceptableAdvisory{}, err.Causes[0])

}
