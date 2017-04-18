package sous

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOwnerSet(t *testing.T) {
	set := NewOwnerSet("one")
	set.Add("two")
	set.Add("one")
	slice := set.Slice()
	assert.Len(t, slice, 2)
	assert.Contains(t, slice, "one")
	assert.Contains(t, slice, "two")
	assert.True(t, set.Equal(NewOwnerSet("two", "one")))
}
