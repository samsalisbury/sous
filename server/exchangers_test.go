package server

import (
	"testing"

	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
)

func TestResourcesFulfillInterfaces(t *testing.T) {
	assert.Implements(t, (*restful.Getable)(nil), newGDMResource(ServerContext{}))
	assert.Implements(t, (*restful.Putable)(nil), newGDMResource(ServerContext{}))

	assert.Implements(t, (*restful.Getable)(nil), newStateDefResource(ServerContext{}))

	assert.Implements(t, (*restful.Getable)(nil), newManifestResource(ServerContext{}))
	assert.Implements(t, (*restful.Putable)(nil), newManifestResource(ServerContext{}))
	assert.Implements(t, (*restful.Deleteable)(nil), newManifestResource(ServerContext{}))

	assert.Implements(t, (*restful.Putable)(nil), newArtifactResource(ServerContext{}))

	assert.Implements(t, (*restful.Getable)(nil), newServerListResource(ServerContext{}))
	assert.Implements(t, (*restful.Putable)(nil), newServerListResource(ServerContext{}))

	assert.Implements(t, (*restful.Getable)(nil), newStatusResource(ServerContext{}))
}
