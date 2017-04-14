package test_with_docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanMakeAnAgent(t *testing.T) {
	assert := assert.New(t)

	agent, err := NewAgent()
	assert.NoError(err)
	assert.NotNil(agent)
}
