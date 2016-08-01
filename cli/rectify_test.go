package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPredicateBuilder(t *testing.T) {
	assert := assert.New(t)

	f := rectifyFlags{}
	assert.Nil(f.buildPredicate())
}
