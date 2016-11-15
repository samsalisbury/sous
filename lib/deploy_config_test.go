package sous

import (
	"testing"

	"github.com/nyarly/testify/assert"
)

func TestValidateRepair(t *testing.T) {
	dc := DeployConfig{
		Volumes:   Volumes{nil, &Volume{}},
		Resources: make(Resources),
	}
	dc.Resources["cpus"] = "0.25"
	dc.Resources["memory"] = "356"
	dc.Resources["ports"] = "2"

	assert.Len(t, dc.Volumes, 2)
	flaws := dc.Validate(DeployConfig{})
	assert.Len(t, flaws, 1)
	fs, es := RepairAll(flaws)
	assert.Len(t, fs, 0)
	assert.Len(t, es, 0)
	assert.Len(t, dc.Volumes, 1)
}
