package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testFunc() {
}

func TestRetrieveMetaData_outsideFunc(t *testing.T) {

	name, uuid := RetrieveMetaData(testFunc)

	assert.Equal(t, name, "github.com/opentable/sous/util/logging.testFunc")
	assert.True(t, len(uuid) > 0)
}

func TestRetrieveMetaData_insideFunc(t *testing.T) {

	f := func() {
	}

	name, uuid := RetrieveMetaData(f)

	assert.Equal(t, name, "github.com/opentable/sous/util/logging.TestRetrieveMetaData_insideFunc.func1")
	assert.True(t, len(uuid) > 0)
}
