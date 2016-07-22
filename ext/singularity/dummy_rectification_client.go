package singularity

import (
	"log"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
)

type (
	// DummyRectificationClient implements RectificationClient but doesn't act on the Mesos scheduler;
	// instead it collects the changes that would be performed and options
	DummyRectificationClient struct {
		logger    *log.Logger
		nameCache sous.Registry
		created   []dummyRequest
		deployed  []dummyDeploy
		scaled    []dummyScale
		deleted   []dummyDelete
	}

	dummyDeploy struct {
		cluster   string
		depID     string
		reqID     string
		imageName string
		res       sous.Resources
		e         sous.Env
		vols      sous.Volumes
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

	dummyDelete struct {
		cluster, reqid, message string
	}
)

// NewDummyRectificationClient builds a new DummyRectificationClient
func NewDummyRectificationClient(nc sous.Registry) *DummyRectificationClient {
	return &DummyRectificationClient{nameCache: nc}
}

// SetLogger sets the logger for the client
func (t *DummyRectificationClient) SetLogger(l *log.Logger) {
	l.Println("dummy begin")
	t.logger = l
}

func (t *DummyRectificationClient) log(v ...interface{}) {
	if t.logger != nil {
		t.logger.Print(v...)
	}
}

func (t *DummyRectificationClient) logf(f string, v ...interface{}) {
	if t.logger != nil {
		t.logger.Printf(f, v...)
	}
}

// Deploy implements part of the RectificationClient interface
func (t *DummyRectificationClient) Deploy(
	cluster, depID, reqID, imageName string, res sous.Resources, e sous.Env, vols sous.Volumes) error {
	t.logf("Deploying instance %s %s %s %s %v %v %v", cluster, depID, reqID, imageName, res, e, vols)
	t.deployed = append(t.deployed, dummyDeploy{cluster, depID, reqID, imageName, res, e, vols})
	return nil
}

// PostRequest (cluster, request id, instance count)
func (t *DummyRectificationClient) PostRequest(
	cluster, id string, count int) error {
	t.logf("Creating application %s %s %d", cluster, id, count)
	t.created = append(t.created, dummyRequest{cluster, id, count})
	return nil
}

//Scale (cluster url, request id, instance count, message)
func (t *DummyRectificationClient) Scale(
	cluster, reqid string, count int, message string) error {
	t.logf("Scaling %s %s %d %s", cluster, reqid, count, message)
	t.scaled = append(t.scaled, dummyScale{cluster, reqid, count, message})
	return nil
}

// DeleteRequest (cluster url, request id, instance count, message)
func (t *DummyRectificationClient) DeleteRequest(
	cluster, reqid, message string) error {
	t.logf("Deleting application %s %s %s", cluster, reqid, message)
	t.deleted = append(t.deleted, dummyDelete{cluster, reqid, message})
	return nil
}

// ImageLabels gets the labels for an image name
func (t *DummyRectificationClient) ImageLabels(in string) (map[string]string, error) {
	a := docker.DockerBuildArtifact(in)
	sv, err := t.nameCache.GetSourceID(a)
	if err != nil {
		return map[string]string{}, nil
	}

	return docker.Labels(sv), nil
}
