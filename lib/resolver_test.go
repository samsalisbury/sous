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
	missing := Deployment{ClusterName: `x`, SourceID: svOne}
	rejected := Deployment{ClusterName: `x`, SourceID: svTwo}
	gdm := MakeDeployments(2)
	gdm.Add(&missing)
	gdm.Add(&rejected)

	dr.FeedArtifact(nil, fmt.Errorf("dummy error"))
	dr.FeedArtifact(&BuildArtifact{"ot-docker/one", "docker", []Quality{{"ephemeral_tag", "advisory"}}}, nil)

	err := errors.Cause(guardImages(dr, gdm)).(*ResolveErrors)
	assert.Error(err)
	require.Len(err.Causes, 2)
}
