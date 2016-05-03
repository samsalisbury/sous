package sous

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type (
	TestRectClient struct {
		created  []dummyRequest
		deployed []dummyDeploy
		scaled   []dummyScale
	}

	dummyDeploy struct {
		cluster   string
		depID     string
		reqID     string
		imageName string
		res       Resources
	}

	dummyRequest struct {
		cluster string
		id      string
		count   int
	}

	dummyScale struct {
		cluster, reqid string
		count          int
		message        string
	}
)

func NewTRC() TestRectClient {
	return TestRectClient{
		created: []dummyRequest{},
	}
}

func (t *TestRectClient) Deploy(cluster string, depID string, reqID string, imageName string, res Resources) error {
	t.deployed = append(t.deployed, dummyDeploy{cluster, depID, reqID, imageName, res})
	return nil
}

// PostRequest(cluster, request id, instance count)
func (t *TestRectClient) PostRequest(cluster string, id string, count int) error {
	t.created = append(t.created, dummyRequest{cluster, id, count})
	return nil
}

//Scale(cluster url, request id, instance count, message)
func (t *TestRectClient) Scale(cluster, reqid string, count int, message string) error {
	t.scaled = append(t.scaled, dummyScale{cluster, reqid, count, message})
	return nil
}

//ImageName finds or guesses a docker image name for a Deployment
func (t *TestRectClient) ImageName(d *Deployment) string {
	return d.String()
}

func TestCreates(t *testing.T) {
	assert := assert.New(t)

	chanset := NewDiffChans(1)
	client := TestRectClient{}

	done := Rectify(chanset, &client)

	created := Deployment{}
	chanset.Created <- created

	chanset.Close()

	<-done
	assert.Len(client.created, 1)
}
