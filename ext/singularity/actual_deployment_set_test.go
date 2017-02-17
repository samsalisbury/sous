package singularity

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"strings"
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/swaggering"
	"github.com/samsalisbury/semv"
)

// Requester copied from swaggering for reference.

// Requester defines the interface that Swaggering uses to
// make actual HTTP requests of the API server
type Requester interface {
	// DTORequest performs an HTTP request and populates a DTO based on the response
	DTORequest(dto swaggering.DTO, method, path string, pathParams, queryParams swaggering.URLParams, body ...swaggering.DTO) error

	// Request performs an HTTP request and returns the body of the response
	Request(method, path string, pathParams, queryParams swaggering.URLParams, body ...swaggering.DTO) (io.ReadCloser, error)
}

type TestGETRequester map[string]swaggering.DTO

func (tgr TestGETRequester) Request(method, path string, pathParams, queryParams swaggering.URLParams, body ...swaggering.DTO) (io.ReadCloser, error) {
	panic("Not implemented.")
}

func (tgr TestGETRequester) DTORequest(dto swaggering.DTO, method, path string, pathParams, queryParams swaggering.URLParams, body ...swaggering.DTO) error {

	// Turn path into a text/template string.
	path = strings.Replace(path, "{", "{{.", -1)
	path = strings.Replace(path, "}", "}}", -1)
	// Populate it with pathParams.
	var t = template.Must(template.New("url").Parse(path))
	pathWriter := &bytes.Buffer{}
	t.Execute(pathWriter, pathParams)
	path = pathWriter.String()

	d, ok := tgr[path]
	if !ok {
		return fmt.Errorf("no DTO at path %q", path)
	}
	return dto.Absorb(d)
}

func TestGetDepSetWorks(t *testing.T) {
	assert := assert.New(t)

	baseURL := "http://test-singularity.org/"

	reg := sous.NewDummyRegistry()

	reg.FeedImageLabels(map[string]string{
		"com.opentable.sous.repo_url":    "github.com/user/project",
		"com.opentable.sous.version":     "1.0.0",
		"com.opentable.sous.revision":    "abc123",
		"com.opentable.sous.repo_offset": "",
	}, nil)

	testReq := &dtos.SingularityRequestParent{
		RequestDeployState: &dtos.SingularityRequestDeployState{
			ActiveDeploy: &dtos.SingularityDeployMarker{
				DeployId:  "testdep",
				RequestId: "testreq",
			},
		},
		Request: &dtos.SingularityRequest{
			Id:          "testreq",
			RequestType: dtos.SingularityRequestRequestTypeSERVICE,
			Owners:      swaggering.StringList{"jlester@opentable.com"},
		},
	}

	testDep := &dtos.SingularityDeployHistory{
		Deploy: &dtos.SingularityDeploy{
			Id: "testdep",
			ContainerInfo: &dtos.SingularityContainerInfo{
				Type: dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
				Docker: &dtos.SingularityDockerInfo{
					Image: "some-docker-image",
				},
				Volumes: dtos.SingularityVolumeList{
					&dtos.SingularityVolume{
						HostPath:      "/onhost",
						ContainerPath: "/indocker",
						Mode:          dtos.SingularityVolumeSingularityDockerVolumeModeRW,
					},
				},
			},
			Resources: &dtos.Resources{},
		},
	}

	requester := TestGETRequester{
		"/api/requests":                               &dtos.SingularityRequestParentList{testReq},
		"/api/requests/request/testreq":               testReq,
		"/api/history/request/testreq/deploy/testdep": testDep,
	}
	client := &singularity.Client{Requester: requester}

	dep := Deployer{
		Registry: reg,
		Client:   client,
		Cluster:  sous.Cluster{Name: "cluster1", BaseURL: baseURL},
	}

	res, err := dep.RunningDeployments()
	if !assert.NoError(err) {
		t.FailNow()
	}
	if !assert.NotNil(res) {
		t.FailNow()
	}

	actual := res.Snapshot()

	assert.Len(actual, 1)

	t.Logf("% #v", res.Snapshot())

	expectedDID := sous.DeployID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: "github.com/user/project",
				Dir:  "",
			},
			Flavor: "",
		},
		Cluster: "cluster1",
	}

	actualDS, ok := actual[expectedDID]
	if !ok {
		var actualDID sous.DeployID
		for actualDID = range actual {
			break
		}
		t.Fatalf("Got DeployID %q; want DeployID %q", actualDID, expectedDID)
	}

	expectedDS := sous.DeployState{
		Deployment: sous.Deployment{
			Kind: sous.ManifestKindService,
			SourceID: sous.SourceID{
				Location: sous.SourceLocation{
					Repo: "github.com/user/project",
					Dir:  "",
				},
				Version: semv.MustParse("1"),
			},
			Flavor:      "",
			ClusterName: "cluster1",
			//Cluster:     cluster,
			DeployConfig: sous.DeployConfig{
				Resources: sous.Resources{
					"cpus":   "0",
					"memory": "0",
					"ports":  "0",
				},
				Volumes: sous.Volumes{
					&sous.Volume{
						Host:      "/onhost",
						Container: "/indocker",
						Mode:      sous.ReadWrite,
					},
				},
			},
		},
		Status: sous.DeployStatusActive,
	}

	different, diffs := actualDS.Diff(&expectedDS)
	if different {
		t.Fatalf("deploy state not as expected: % #v", diffs)
	}

}
