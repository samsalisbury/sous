package docker_registry

import (
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func TestRegistries(t *testing.T) {
	assert := assert.New(t)

	rs := NewRegistries()
	r := &registry{}
	assert.NoError(rs.AddRegistry("x", r))
	assert.Equal(rs.GetRegistry("x"), r)
	assert.NoError(rs.DeleteRegistry("x"))
	assert.Nil(rs.GetRegistry("x"))
}

// This test is terrible, but the current design of the client is hard to test
func TestNewClient(t *testing.T) {
	assert := assert.New(t)

	c := NewClient(logging.SilentLogSet())
	assert.NotNil(c)
	c.Cancel()
}
