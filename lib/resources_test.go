package sous

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRepairResources(t *testing.T) {
	empty := make(Resources)

	flaws := empty.Validate()
	assert.Len(t, flaws, 3)

	flaws, es := RepairAll(flaws)
	assert.Len(t, flaws, 0)
	assert.Len(t, es, 0)
	assert.Equal(t, empty["cpus"], "0.1")
	assert.Equal(t, empty["memory"], "100")
	assert.Equal(t, empty["ports"], "1")
}
