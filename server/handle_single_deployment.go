package server

import sous "github.com/opentable/sous/lib"

type (
	PUTSingleDeploymentHandler struct {
		DeploymentID      sous.DeploymentID
		DeploymentIDError error
		Manifest          sous.Manifest
		ManifestError     error
	}
	singleDeploymentResponse struct{}
)

func (psd *PUTSingleDeploymentHandler) Exchange() (interface{}, int) {
	return singleDeploymentResponse{}, 404
}
