package sous

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsTransientResolveError(t *testing.T) {
	assert := assert.New(t)

	assert.False(IsTransientResolveError(fmt.Errorf("Hi!")))
	assert.False(IsTransientResolveError(&UnacceptableAdvisory{}))
	assert.False(IsTransientResolveError(errors.Wrap(&MissingImageNameError{}, "wrapped")))
	assert.True(IsTransientResolveError(&CreateError{}))
	assert.True(IsTransientResolveError(errors.Wrap(&CreateError{}, "even if wrapped")))
}
