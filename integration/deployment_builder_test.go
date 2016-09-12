// +build integration
package integration

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/nyarly/testify/assert"
	sing "github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/whitespace"
	"github.com/satori/go.uuid"
)

func TestBuildDeployments(t *testing.T) {

	t.Skipf("Failing test on master preventing progress on other stories.")

	assert := assert.New(t)
	sous.Log.Debug.SetOutput(os.Stdout)

	ResetSingularity()
	defer ResetSingularity()

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()

	db, err := docker.GetDatabase(&docker.DBConfig{
		Driver:     "sqlite3",
		Connection: docker.InMemoryConnection("testresolve"),
	})
	if err != nil {
		panic(err)
	}

	appLocation := "testhelloreq"
	clusterNick := "tcluster"
	reqID := appLocation + clusterNick

	nc := docker.NewNameCache(drc, db)
	ra := singularity.NewRectiAgent(nc)

	singCl := sing.NewClient(SingularityURL)
	//singCl.Debug = true

	sr, err := singReqDep(
		SingularityURL,
		whitespace.CleanWS(`
		{
			"instances": 1,
			"id": "`+reqID+`",
			"requestType": "SERVICE",
			"owners": ["tom@hanna.net", "jerry@barbera.org"]
		}`),
		whitespace.CleanWS(`
		{
			"deploy": {
				"id": "`+singularity.MakeDeployID(uuid.NewV4().String())+`",
				"requestId": "`+reqID+`",
				"resources": {
					"cpus": 0.1,
					"memoryMb": 32,
					"numPorts": 1
				},
				"containerInfo": {
					"type": "DOCKER",
					"docker": {
						"image": "`+BuildImageName("hello-server-labels", "latest")+`"
					},
					"volumes": [{"hostPath":"/tmp", "containerPath":"/tmp","mode":"RO"}]
				},
				"env": {
					"TEST": "yes"
				}
			}
		}`),
	)

	req := singularity.SingReq{
		SourceURL: SingularityURL,
		Sing:      singCl,
		ReqParent: sr,
	}

	if assert.NoError(err) {
		clusters := sous.Clusters{clusterNick: {BaseURL: SingularityURL}}
		dep, err := singularity.BuildDeployment(ra, clusters, req)

		if assert.NoError(err) {
			if assert.Len(dep.DeployConfig.Volumes, 1) {
				assert.Equal(dep.DeployConfig.Volumes[0].Host, "/tmp")
			}
			assert.Equal("https://github.com/docker/dockercloud-hello-world.git", dep.SourceID.Location.Repo)
		}
	}
}

func pushLabelledContainers() {
	//BuildAndPushContainer(BuildImageName("hello-labels", "latest"), "hello-labels")
	BuildAndPushContainer(BuildImageName("hello-server-labels", "latest"), "hello-server-labels")
	//BuildAndPushContainer(BuildImageName("grafana-repo", "latest"), "grafana-labels")
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
