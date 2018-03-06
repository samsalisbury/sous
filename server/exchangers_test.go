package server

import (
	"testing"

	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
)

func TestResourcesFulfillInterfaces(t *testing.T) {
	assert.Implements(t, (*restful.Getable)(nil), newGDMResource(ComponentLocator{}))
	assert.Implements(t, (*restful.Putable)(nil), newGDMResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Getable)(nil), newStateDeploymentResource(ComponentLocator{}))
	assert.Implements(t, (*restful.Putable)(nil), newStateDeploymentResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Getable)(nil), newStateDefResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Getable)(nil), newManifestResource(ComponentLocator{}))
	assert.Implements(t, (*restful.Putable)(nil), newManifestResource(ComponentLocator{}))
	assert.Implements(t, (*restful.Deleteable)(nil), newManifestResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Putable)(nil), newArtifactResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Getable)(nil), newServerListResource(ComponentLocator{}))
	assert.Implements(t, (*restful.Putable)(nil), newServerListResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Getable)(nil), newStatusResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Getable)(nil), newHealthResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Getable)(nil), newAllDeployQueuesResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Getable)(nil), newDeployQueueResource(ComponentLocator{}))

	assert.Implements(t, (*restful.Getable)(nil), newR11nResource(ComponentLocator{}))
}
