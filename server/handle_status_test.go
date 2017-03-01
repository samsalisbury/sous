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

	logSet := &sous.LogSet{
		Vomit: log.New(os.Stderr, "", log.LstdFlags),
	}
	autoResolver := sous.NewAutoResolver(nil, nil, nil)
	autoResolver.GDM = sous.NewDeployments()
	th := &StatusHandler{
		AutoResolver:  autoResolver,
		Log:           logSet,
		ResolveFilter: &sous.ResolveFilter{},
	}
	data, status := th.Exchange()
	assert.Equal(status, 200)
	assert.Len(data.(statusData).Deployments, 0)
}
