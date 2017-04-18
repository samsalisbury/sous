package sous

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_String(t *testing.T) {
	assert.Equal(t, User{}.String(), "")
	assert.Equal(t,
		User{
			Name:  "Judson",
			Email: "jlester@opentable.com",
		}.String(),
		"Judson <jlester@opentable.com>")
}

func TestUser_Complete(t *testing.T) {
	assert.False(t, User{}.Complete())
	assert.False(t, User{Name: "x"}.Complete())
	assert.False(t, User{Email: "y"}.Complete())
	assert.True(t, User{Name: "x", Email: "y"}.Complete())
}
