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
		nameCache sous.Builder
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

	// DummyNameCache implements the Builder interface by returning a
	// computed image name for a given source version
	DummyNameCache struct {
	}
)

// NewDummyRectificationClient builds a new DummyRectificationClient
func NewDummyRectificationClient(nc sous.Builder) *DummyRectificationClient {
	return &DummyRectificationClient{nameCache: nc}
}

// TODO: Factor out name cache concept from core sous lib & get rid of this func.
func (t *DummyRectificationClient) GetRunningDeployment([]string) (sous.Deployments, error) {
	return nil, nil
	panic("not implemented")
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

//ImageName finds or guesses a docker image name for a Deployment
func (t *DummyRectificationClient) ImageName(d *sous.Deployment) (string, error) {
	a, err := t.nameCache.GetArtifact(d.SourceVersion)
	if err != nil {
		return "", err
	}
	return a.Name, nil
}

// ImageLabels gets the labels for an image name
func (t *DummyRectificationClient) ImageLabels(in string) (map[string]string, error) {
	a := &sous.BuildArtifact{Name: in}
	sv, err := t.nameCache.GetSourceVersion(a)
	if err != nil {
		return map[string]string{}, nil
	}

	return docker.DockerLabels(sv), nil
}

// NewDummyNameCache builds a new DummyNameCache
func NewDummyNameCache() *DummyNameCache {
	return &DummyNameCache{}
}

// TODO: Factor out name cache concept from core sous lib & get rid of this func.
func (dc *DummyNameCache) Build(*sous.BuildContext, sous.Buildpack, *sous.DetectResult) (*sous.BuildResult, error) {
	return nil, nil
	panic("not implemented")
}

// TODO: Factor out name cache concept from core sous lib & get rid of this func.
func (dc *DummyNameCache) GetArtifact(sv sous.SourceVersion) (*sous.BuildArtifact, error) {
	imageName, err := dc.GetImageName(sv)
	if err != nil {
		return nil, err
	}
	return &sous.BuildArtifact{Name: imageName, Type: "docker"}, nil
}

// GetImageName implements part of the interface for ImageMapper
func (dc *DummyNameCache) GetImageName(sv sous.SourceVersion) (string, error) {
	return sv.String(), nil
}

// GetCanonicalName implements part of the interface for ImageMapper
// It simply returns whatever it was given
func (dc *DummyNameCache) GetCanonicalName(in string) (string, error) {
	return in, nil
}

// Insert implements part of ImageMapper
// it drops the sv/in pair on the floor
func (dc *DummyNameCache) Insert(sv sous.SourceVersion, in, etag string) error {
	return nil
}

// GetSourceVersion implements part of ImageMapper
func (dc *DummyNameCache) GetSourceVersion(*sous.BuildArtifact) (sous.SourceVersion, error) {
	return sous.SourceVersion{}, nil
}
