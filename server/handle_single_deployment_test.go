package server

import (
	"io"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
)

func TestPUTSingleDeploymentHandler_Exchange(t *testing.T) {

	makeRequestBody := func() io.ReadCloser {
		manifest := sous.Manifest{
			Deployments: sous.DeploySpecs{
				"cluster1": sous.DeploySpec{
					DeployConfig: sous.DeployConfig{
						Resources: map[string]string{
							"cpus":   "0.1",
							"memory": "32",
						},
						NumInstances: 2,
					},
					Version: semv.MustParse("1"),
				},
			},
		}
		return nil
	}

	h := &PUTSingleDeploymentHandler{
		QueueSet:        sous.NewR11nQueueSet(),
		RequestBody:     makeRequestBody(),
		DeploymentID:    deploy1,
		DeploymentIDErr: nil,
	}

}
