package docker

import (
	"io/ioutil"
	"os"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func TestDbInsertNoDump(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()
	host := "docker.repo.io"
	nc, err := NewNameCache(host, dc, logging.SilentLogSet(), sous.SetupDB(t))
	defer sous.ReleaseDB(t)

	assert.NoError(err)
	assert.NotNil(nc, "should be populated")
	//assert.Fail("broke")

	base := "wacky"
	tag := "1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	cn := base + "@" + digest
	//in := base + ":" + tag

	sid := sous.MustNewSourceID("https://github.com/opentable/wacky", "", tag)

	rescueStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err = nc.dbInsert(sid, cn, "etag", nil, nil)
	assert.NoError(err, "insert should succeed")

	w.Close()
	stderr, _ := ioutil.ReadAll(r)

	assert.NotContains(string(stderr), "select *", "shouldn't contain any select statements")
	os.Stderr = rescueStderr
}
