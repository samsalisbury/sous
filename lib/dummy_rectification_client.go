package sous

import "log"

type (
	// DummyRectificationClient implements RectificationClient but doesn't act on the Mesos scheduler;
	// instead it collects the changes that would be performed and options
	DummyRectificationClient struct {
		logger    *log.Logger
		nameCache Registry
		Created   []dummyRequest
		Deployed  []dummyDeploy
		Scaled    []dummyScale
		Deleted   []dummyDelete
	}

	dummyDeploy struct {
		Cluster   string
		DepID     string
		ReqID     string
		ImageName string
		Res       Resources
		E         Env
		Vols      Volumes
	}

	dummyRequest struct {
		Cluster string
		ID      string
		Count   int
	}

	dummyScale struct {
		Cluster, Reqid string
		Count          int
		Message        string
	}

	dummyDelete struct {
		Cluster, Reqid, Message string
	}
)

// NewDummyRectificationClient builds a new DummyRectificationClient
func NewDummyRectificationClient(nc Registry) *DummyRectificationClient {
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
	cluster, depID, reqID, imageName string, res Resources, e Env, vols Volumes) error {
	t.logf("Deploying instance %s %s %s %s %v %v %v", cluster, depID, reqID, imageName, res, e, vols)
	t.Deployed = append(t.Deployed, dummyDeploy{cluster, depID, reqID, imageName, res, e, vols})
	return nil
}

// PostRequest (cluster, request id, instance count)
func (t *DummyRectificationClient) PostRequest(
	cluster, id string, count int) error {
	t.logf("Creating application %s %s %d", cluster, id, count)
	t.Created = append(t.Created, dummyRequest{cluster, id, count})
	return nil
}

//Scale (cluster url, request id, instance count, message)
func (t *DummyRectificationClient) Scale(
	cluster, reqid string, count int, message string) error {
	t.logf("Scaling %s %s %d %s", cluster, reqid, count, message)
	t.Scaled = append(t.Scaled, dummyScale{cluster, reqid, count, message})
	return nil
}

// DeleteRequest (cluster url, request id, instance count, message)
func (t *DummyRectificationClient) DeleteRequest(
	cluster, reqid, message string) error {
	t.logf("Deleting application %s %s %s", cluster, reqid, message)
	t.Deleted = append(t.Deleted, dummyDelete{cluster, reqid, message})
	return nil
}
