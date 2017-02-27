package server

import (
	"log"
	"os"
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
		Log: &sous.LogSet{
			Vomit: log.New(os.Stderr, "", log.LstdFlags),
		},
	}
	data, status := th.Exchange()
	assert.Equal(status, 200)
	assert.Len(data.(statusData).Deployments, 0)
}
