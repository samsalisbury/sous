package sous

import (
	"testing"

	"github.com/nyarly/testify/assert"
)

func TestValidateRepair(t *testing.T) {
	dc := DeployConfig{
		Volumes: Volumes{nil, &Volume{}},
	}

	assert.Len(t, dc.Volumes, 2)
	flaws := dc.Validate()
	assert.Len(t, flaws, 1)
	fs, es := RepairAll(flaws)
	assert.Len(t, fs, 0)
	assert.Len(t, es, 0)
	assert.Len(t, dc.Volumes, 1)
}
