package restfultest

import (
	"testing"

	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
)

func TestClientSpyImplementsInterface(t *testing.T) {
	assert.Implements(t, (*restful.HTTPClient)(nil), &HTTPClientSpy{})
}
