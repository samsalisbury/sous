package server

import (
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
)

func TestPUTSingleDeploymentHandler_Exchange(t *testing.T) {

	makeSingleDeploymentBody := func(repo, cluster, flavor string) *singleDeploymentBody {
		manifest := sous.Manifest{
			Source: sous.SourceLocation{
				Repo: "",
				Dir:  "",
			},
			Flavor: "",
			Owners: nil,
			Kind:   "",
			Deployments: map[string]sous.DeploySpec{
				"": {
					DeployConfig: sous.DeployConfig{
						Resources: map[string]string{
							"": "",
						},
						Metadata: map[string]string{
							"": "",
						},
						Env: map[string]string{
							"": "",
						},
						NumInstances: 0,
						Volumes:      nil,
						Startup: sous.Startup{
							SkipCheck:                 false,
							ConnectDelay:              0,
							Timeout:                   0,
							ConnectInterval:           0,
							CheckReadyProtocol:        "",
							CheckReadyURIPath:         "",
							CheckReadyPortIndex:       0,
							CheckReadyFailureStatuses: nil,
							CheckReadyURITimeout:      0,
							CheckReadyInterval:        0,
							CheckReadyRetries:         0,
						},
						Schedule: "",
					},
					Version: semv.MustParse("1"),
				},
			},
		}
		return &singleDeploymentBody{
			ManifestHeader: manifest,
			DeploymentID:   sous.DeploymentID{},
			DeploySpec:     sous.DeploySpec{},
		}
	}

	testCases := []struct {
		// BodyAndID is a function that generates a body and an ID.
		// We expect that if response.DeploymentID == id and the server is
		// configured to service requests from the corresponding cluster,
		// the GDM should be updated and a new R11n enqueued.
		//
		// The body is sent as the PUT body of the request.
		// We expect that the same body is returned on success.
		BodyAndID func() (*singleDeploymentBody, sous.DeploymentID)
		// WantStatus is the expected HTTP status for this request.
		WantStatus int
	}{
		{
			BodyAndID: func() (*singleDeploymentBody, sous.DeploymentID) {
				response := makeSingleDeploymentBody("github.com/user1/repo1", "cluster1", "flavor1")
				return response, response.DeploymentID
			},
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {

			sent, did := tc.BodyAndID()

			psd := PUTSingleDeploymentHandler{
				DeploymentID: did,
				Body:         sent,
			}

			received, gotStatus := psd.Exchange()

			got, ok := received.(*singleDeploymentBody)

			if !ok {
				t.Fatalf("got a %T; want a %T", received, got)
			}

			wantStatus := 404
			if gotStatus != wantStatus {
				t.Errorf("got status %d; want %d", gotStatus, wantStatus)
			}
		})
	}

}
