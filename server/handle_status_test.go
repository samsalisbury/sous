package server

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/sous/lib"
)

func TestHandlesStatusGet(t *testing.T) {
	assert := assert.New(t)

	th := &StatusHandler{
		AutoResolver: &sous.AutoResolver{
			GDM: sous.NewDeployments(),
		},
	}
	data, status := th.Exchange()
	assert.Equal(status, 200)
	assert.Len(data.(statusData).Deployments, 0)
}
