package sous

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRepair(t *testing.T) {
	dc := DeployConfig{
		Volumes:   Volumes{nil, &Volume{}},
		Resources: make(Resources),
		Startup: Startup{

			SkipTest: true,
		},
	}
	dc.Resources["cpus"] = "0.25"
	dc.Resources["memory"] = "356"
	dc.Resources["ports"] = "2"

	t.Log(dc.Startup)

	assert.Len(t, dc.Volumes, 2)
	flaws := dc.Validate()
	assert.Len(t, flaws, 1)
	fs, es := RepairAll(flaws)
	assert.Len(t, fs, 0)
	assert.Len(t, es, 0)
	assert.Len(t, dc.Volumes, 1)
}
