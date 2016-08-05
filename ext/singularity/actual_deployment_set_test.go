package singularity

import (
	"testing"

	"github.com/opentable/go-singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/swaggering"
	"github.com/stretchr/testify/assert"
)

func TestGetDepSetWorks(t *testing.T) {
	assert := assert.New(t)

	whip := make(map[string]swaggering.DummyControl)

	reg := sous.NewDummyRegistry()
	client := NewDummyRectificationClient(reg)
	dep := deployer{reg, client,
		func(url string) *singularity.Client {
			cl, co := singularity.NewDummyClient(url)
			whip[url] = co
			return cl
		},
	}

	res, err := dep.GetRunningDeployment(map[string]string{`test`: `http://test-singularity.org/`})
	assert.NoError(err)
	assert.NotNil(res)
}
