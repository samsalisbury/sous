package test

import (
	"bytes"
	"net/http"
	"testing"
	"whitespace"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/stretchr/testify/assert"
)

func TestBuildDeployments(t *testing.T) {
	assert := assert.New(t)

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			singularityURL: sous.Cluster{
				BaseURL: singularityURL,
			},
		},
	}
	repoOne := "https://github.com/opentable/one.git"
	repoTwo := "https://github.com/opentable/two.git"
	repoThree := "https://github.com/opentable/three.git"

	drc := docker_registry.NewClient()
	drc.BecomeFoolishlyTrusting()

	nc := sous.NewNameCache(drc, "sqlite3", sous.InMemoryConnection("testresolve"))
	ra := sous.NewRectiAgent(nc)

	singReqDep(
		singularityURL,
		whitespace.CleanWS(`
		{
			"instances": 1,
			"id": "test-grafana-request",
			"requestType": "SERVICE",
		}`),
		whitespace.CleanWS(`
		{
			id: "test-grafana-deploy",
			requestId: "test-grafana-request",
			"resources": {
				"cpu": 0.1,
				"mem": 32,
				"ports": 1,
			},
			containerInfo: {

			},
			env: {
				"TEST": "yes"
			},


		}
			`),
	)

	resetSingularity()
}

func pushLabelledContainers() {
	//buildAndPushContainer(buildImageName("hello-labels", "latest"), "hello-labels")
	//buildAndPushContainer(buildImageName("hello-server-labels", "latest"), "hello-server-labels")
	buildAndPushContainer(buildImageName("grafana-repo", "latest"), "grafana-labels")
}

func singReqDep(url, ryaml, dyaml string) error {
	h := &http.Client{}
	ru := url + `/api/requests`
	du := url + `/api/deploys`

	_, err := h.Post(ru, `application/json`, bytes.NewBufferString(ryaml))
	if err != nil {
		return err
	}

	_, err = h.Post(du, `application/json`, bytes.NewBufferString(dyaml))
	if err != nil {
		return err
	}

	return nil
}
