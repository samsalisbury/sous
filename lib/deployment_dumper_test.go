package sous

import (
	"bytes"
	"testing"

	"github.com/nyarly/testify/assert"
)

func TestDeploymentDumper(t *testing.T) {
	assert := assert.New(t)

	io := &bytes.Buffer{}
	ds := NewDeployments()
	ds.Add(&Deployment{ClusterName: "andromeda"})

	DumpDeployments(io, ds)
	assert.Regexp(`andromeda`, io.String())
}
