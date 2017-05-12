package server

import (
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/stretchr/testify/assert"
)

func TestHandlesGDMGet(t *testing.T) {
	assert := assert.New(t)

	th := &GETGDMHandler{&LiveGDM{
		Deployments: sous.NewDeployments(),
	}}
	data, status := th.Exchange()
	assert.Equal(status, 200)
	assert.Len(data.(gdmWrapper).Deployments, 0)
}
