package sous

import "log"

type (
	// DummyRectificationClient implements RectificationClient but doesn't act on the Mesos scheduler;
	// instead it collects the changes that would be performed and options
	DummyRectificationClient struct {
		logger    *log.Logger
		nameCache ImageMapper
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
		res       Resources
		e         Env
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

	// DummyNameCache implements the ImageMapper interface by returning a
	// computed image name for a given source version
	DummyNameCache struct {
	}
)

// NewDummyRectificationClient builds a new DummyRectificationClient
func NewDummyRectificationClient(nc ImageMapper) *DummyRectificationClient {
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
	cluster, depID, reqID, imageName string, res Resources, e Env) error {
	t.logf("Deploying instance %s %s %s %s %v %v", cluster, depID, reqID, imageName, res, e)
	t.deployed = append(t.deployed, dummyDeploy{cluster, depID, reqID, imageName, res, e})
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

//ImageName finds or guesses a docker image name for a Deployment
func (t *DummyRectificationClient) ImageName(d *Deployment) (string, error) {
	return t.nameCache.GetImageName(d.SourceVersion)
}

// NewDummyNameCache builds a new DummyNameCache
func NewDummyNameCache() *DummyNameCache {
	return &DummyNameCache{}
}

// GetImageName implements part of the interface for ImageMapper
func (dc *DummyNameCache) GetImageName(sv SourceVersion) (string, error) {
	return sv.String(), nil
}

// GetCanonicalName implements part of the interface for ImageMapper
// It simply returns whatever it was given
func (dc *DummyNameCache) GetCanonicalName(in string) (string, error) {
	return in, nil
}

// Insert implements part of ImageMapper
// it drops the sv/in pair on the floor
func (dc *DummyNameCache) Insert(sv SourceVersion, in, etag string) error {
	return nil
}
