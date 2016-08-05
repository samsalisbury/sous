package singularity

import (
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/stretchr/testify/assert"
)

func TestGetDepSetWorks(t *testing.T) {
	assert := assert.New(t)

	reg := sous.NewDummyRegistry()
	client := NewDummyRectificationClient(reg)
	dep := NewDeployer(reg, client)

	res, err := dep.GetRunningDeployment(map[string]string{`test`: `http://test-singularity.org/`})
	assert.NoError(err)
	assert.NotNil(res)
}
