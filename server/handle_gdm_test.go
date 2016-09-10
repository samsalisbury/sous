package server

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

func TestHandlesGDMGet(t *testing.T) {
	assert := assert.New(t)

	th := &GDMHandler{graph.CurrentGDM{
		Deployments: sous.NewDeployments(),
	}}
	data, status := th.Exchange()
	assert.Equal(status, 200)
	assert.Len(data.(gdmWrapper).Deployments, 0)

}
