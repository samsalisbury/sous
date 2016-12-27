package server

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

func TestHandlesStatusGet(t *testing.T) {
	assert := assert.New(t)

	th := &StatusHandler{
		GDM: graph.CurrentGDM{
			Deployments: sous.NewDeployments(),
		},
		AutoResolver: &sous.AutoResolver{},
	}
	data, status := th.Exchange()
	assert.Equal(status, 200)
	assert.Len(data.(StatusResource).Deployments, 0)
}
