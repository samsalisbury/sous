package server

import sous "github.com/opentable/sous/lib"

type (
	// PUTSingleDeploymentHandler handles manifests containing single deployment
	// specs. See Exchange method for more details.
	PUTSingleDeploymentHandler struct {
		DeploymentID    sous.DeploymentID
		DeploymentIDErr error
		Body            *singleDeploymentBody
		BodyErr         error
	}

	// singleDeploymentBody is the response struct returned from handlers
	// of HTTP methods of a SingleDeploymentResource.
	singleDeploymentBody struct {
		DeploymentID   sous.DeploymentID
		DeploySpec     sous.DeploySpec
		ManifestHeader sous.Manifest
	}
)

// Exchange triggers a deployment action when receiving
// a Manifest containing a deployment matching DeploymentID that differs
// from the current actual deployment set. It first writes the new
// deployment spec to the GDM.
func (psd *PUTSingleDeploymentHandler) Exchange() (interface{}, int) {
	return nil, 404
}
