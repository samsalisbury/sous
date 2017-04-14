package whitespace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanWS(t *testing.T) {
	assert.Equal(t, CleanWS(" x"), "x")
	assert.Equal(t, CleanWS(`
		x
		x
		x`), "x\nx\nx")

}
