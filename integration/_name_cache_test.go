// +build integration

package integration

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/util/docker_registry"
)

func TestNameCache(t *testing.T) {
	assert := assert.New(t)

	ResetSingularity()
	defer ResetSingularity()

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()

	db, err := docker.GetDatabase(&docker.DBConfig{
		Driver:     "sqlite3_sous",
		Connection: docker.InMemoryConnection("testnamecache"),
	})
	if err != nil {
		t.Fatal(err)
	}
	nc := docker.NewNameCache("", drc, db)

	repoOne := "https://github.com/opentable/one.git"
	manifest(nc, "opentable/one", "test-one", repoOne, "1.1.1")

	cn, err := nc.GetCanonicalName(BuildImageName("opentable/one", "1.1.1"))
	if err != nil {
		assert.FailNow(err.Error())
	}
	labels, err := drc.LabelsForImageName(cn)

	if assert.NoError(err) {
		assert.Equal("1.1.1", labels[docker.DockerVersionLabel])
	}
}
