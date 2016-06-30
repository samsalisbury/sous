package test

import (
	"os"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/stretchr/testify/assert"
)

func TestNameCache(t *testing.T) {
	assert := assert.New(t)
	sous.Log.Debug.SetOutput(os.Stdout)
	sous.Log.Vomit.SetOutput(os.Stdout)

	resetSingularity()
	defer resetSingularity()

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()

	nc := sous.NewNameCache(drc, "sqlite3", sous.InMemoryConnection("testnamecache"))

	repoOne := "https://github.com/opentable/one.git"
	manifest(nc, "opentable/one", "test-one", repoOne, "1.1.1")

	cn, err := nc.GetCanonicalName(buildImageName("opentable/one", "1.1.1"))
	if err != nil {
		assert.FailNow(err.Error())
	}
	labels, err := drc.LabelsForImageName(cn)

	if assert.NoError(err) {
		assert.Equal("1.1.1", labels[sous.DockerVersionLabel])
	}
}
