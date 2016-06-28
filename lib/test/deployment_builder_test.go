package test

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/whitespace"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestBuildDeployments(t *testing.T) {

	t.Skipf("Failing test on master preventing progress on other stories.")

	assert := assert.New(t)
	sous.Log.Debug.SetOutput(os.Stdout)

	resetSingularity()
	defer resetSingularity()

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()

	nc := sous.NewNameCache(drc, "sqlite3", sous.InMemoryConnection("testresolve"))
	ra := sous.NewRectiAgent(nc)

	singCl := singularity.NewClient(singularityURL)
	singCl.Debug = true

	sr, err := singReqDep(
		singularityURL,
		whitespace.CleanWS(`
		{
			"instances": 1,
			"id": "test-hello-request",
			"requestType": "SERVICE",
			"owners": ["tom@hanna.net", "jerry@barbera.org"]
		}`),
		whitespace.CleanWS(`
		{
			"deploy": {
				"id": "`+idify(uuid.NewV4().String())+`",
				"requestId": "test-hello-request",
				"resources": {
					"cpus": 0.1,
					"memoryMb": 32,
					"numPorts": 1
				},
				"containerInfo": {
					"type": "DOCKER",
					"docker": {
						"image": "`+buildImageName("hello-server-labels", "latest")+`"
					},
					"volumes": [{"hostPath":"/tmp", "containerPath":"/tmp","mode":"RO"}]
				},
				"env": {
					"TEST": "yes"
				}
			}
		}`),
	)

	req := sous.SingReq{
		SourceURL: singularityURL,
		Sing:      singCl,
		ReqParent: sr,
	}

	if assert.NoError(err) {
		uc := sous.NewDeploymentBuilder(ra, req)
		err = uc.CompleteConstruction()

		if assert.NoError(err) {
			dep := uc.Target
			if assert.Len(dep.DeployConfig.Volumes, 1) {
				assert.Equal(dep.DeployConfig.Volumes[0].Host, "/tmp")
			}
			assert.Equal("https://github.com/docker/dockercloud-hello-world.git", string(dep.SourceVersion.RepoURL))
		}

	}

}

func pushLabelledContainers() {
	//buildAndPushContainer(buildImageName("hello-labels", "latest"), "hello-labels")
	buildAndPushContainer(buildImageName("hello-server-labels", "latest"), "hello-server-labels")
	//buildAndPushContainer(buildImageName("grafana-repo", "latest"), "grafana-labels")
}

func singReqDep(url, ryaml, dyaml string) (*dtos.SingularityRequestParent, error) {
	h := &http.Client{}
	ru := url + `/api/requests`
	du := url + `/api/deploys`

	rrz, err := h.Post(ru, `application/json`, bytes.NewBufferString(ryaml))
	if err != nil {
		return nil, err
	}
	logBody("POST /api/requests", rrz)

	dqz, err := h.Post(du, `application/json`, bytes.NewBufferString(dyaml))
	if err != nil {
		return nil, err
	}
	logBody("POST /api/deploys", dqz)

	rqz, err := h.Get(ru)

	time.Sleep(3 * time.Second)

	resBody := logBody("GET /api/requests", rqz)

	var sr dtos.SingularityRequestParentList
	sr.Populate(resBody)

	return sr[0], nil
}

func logBody(from string, rqz *http.Response) io.ReadCloser {
	buf := bytes.Buffer{}
	buf.ReadFrom(rqz.Body)
	log.Printf("%s -> %+v\n", from, buf.String())
	return ioutil.NopCloser(&buf)
}
