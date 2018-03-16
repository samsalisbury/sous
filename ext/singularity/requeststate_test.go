package singularity

import (
	"testing"

	"github.com/nyarly/spies"
	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

func TestStatus(t *testing.T) {
	ls, _ := logging.NewLogSinkSpy()
	dep := &deployer{
		log: ls,
	}

	sing := singClientFixture()
	dep.SetSingularityFactory(func(string) singClient {
		return sing
	})

	reg, rc := sous.NewRegistrySpy()
	rc.MatchMethod("ImageLabels", spies.AnyArgs, map[string]string{
		"com.opentable.sous.repo_url":    "github.com/example/test",
		"com.opentable.sous.version":     "1.2.3",
		"com.opentable.sous.revision":    "",
		"com.opentable.sous.repo_offset": "",
	}, nil)

	clusters := sous.Clusters{
		"cluster-1": &sous.Cluster{BaseURL: "http://sing,example.com"},
	}
	pair := &sous.DeployablePair{
		Post: sous.DeployableFixture(""),
	}

	status, err := dep.Status(reg, clusters, pair)
	if err != nil {
		t.Fatalf("%+#v", err)
	}

	expectedStatus := sous.DeployStatusActive
	if status.Status != expectedStatus {
		t.Errorf("Expected status %q, got %q.", expectedStatus, status.Status)
	}

}

func singClientFixture() singClient {
	req := &dtos.SingularityRequestParent{
		Request: &dtos.SingularityRequest{
			RequestType: dtos.SingularityRequestRequestTypeSERVICE,
		},
		RequestDeployState: &dtos.SingularityRequestDeployState{
			RequestId: "request-id",
		},
	}

	dh := &dtos.SingularityDeployHistory{
		DeployResult: &dtos.SingularityDeployResult{
			DeployState: dtos.SingularityDeployResultDeployStateSUCCEEDED,
		},

		DeployMarker: &dtos.SingularityDeployMarker{},
		Deploy: &dtos.SingularityDeploy{
			Metadata: map[string]string{
				sous.ClusterNameLabel: "cluster-1",
			},
			ContainerInfo: &dtos.SingularityContainerInfo{
				Type: dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
				Docker: &dtos.SingularityDockerInfo{
					Image: "docker-image",
				},
			},
			Resources: &dtos.Resources{},
		},
	}
	dhl := dtos.SingularityDeployHistoryList{dh}

	sing, c := newSingClientSpy()
	c.MatchMethod("GetRequest", spies.AnyArgs, req, nil)
	c.MatchMethod("GetDeploy", spies.AnyArgs, dh, nil)
	c.MatchMethod("GetDeploys", spies.AnyArgs, dhl, nil)

	return sing
}
