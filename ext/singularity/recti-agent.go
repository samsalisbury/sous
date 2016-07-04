package singularity

import (
	"regexp"
	"sync"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/satori/go.uuid"
)

// TODO: notInIDRE is copied from sous/lib. It should only live in this package.
var notInIDRE = regexp.MustCompile(`[-/:]`)

// TODO: idify is copied from sous/lib. It should only live in this package.
func idify(in string) string {
	return notInIDRE.ReplaceAllString(in, "")
}

// RectiAgent is an implementation of the RectificationClient interface
type RectiAgent struct {
	singClients map[string]*singularity.Client
	sync.RWMutex
	nameCache sous.Builder
}

// NewRectiAgent returns a set-up RectiAgent
func NewRectiAgent(b sous.Builder) *RectiAgent {
	return &RectiAgent{
		singClients: make(map[string]*singularity.Client),
		nameCache:   b,
	}
}

// Deploy sends requests to Singularity to make a deployment happen
func (ra *RectiAgent) Deploy(cluster, depID, reqID, dockerImage string,
	r sous.Resources, e sous.Env, vols sous.Volumes) error {
	Log.Debug.Printf("Deploying instance %s %s %s %s %v %v", cluster, depID, reqID, dockerImage, r, e)
	dockerInfo, err := dtos.LoadMap(&dtos.SingularityDockerInfo{}, sous.DTOMap{
		"Image": dockerImage,
	})
	if err != nil {
		return err
	}

	res, err := dtos.LoadMap(&dtos.Resources{}, r.SingMap())
	if err != nil {
		return err
	}

	vs := dtos.SingularityVolumeList{}
	for _, v := range vols {
		sv, err := dtos.LoadMap(&dtos.SingularityVolume{}, sous.DTOMap{
			"ContainerPath": v.Container,
			"HostPath":      v.Host,
			"Mode":          dtos.SingularityVolumeSingularityDockerVolumeMode(string(v.Mode)),
		})
		if err != nil {
			return err
		}
		vs = append(vs, sv.(*dtos.SingularityVolume))
	}

	ci, err := dtos.LoadMap(&dtos.SingularityContainerInfo{}, sous.DTOMap{
		"Type":    dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
		"Docker":  dockerInfo,
		"Volumes": vs,
	})
	if err != nil {
		return err
	}

	dep, err := dtos.LoadMap(&dtos.SingularityDeploy{}, sous.DTOMap{
		"Id":            idify(uuid.NewV4().String()),
		"RequestId":     reqID,
		"Resources":     res,
		"ContainerInfo": ci,
		"Env":           map[string]string(e),
	})
	Log.Debug.Printf("Deploy: %+ v", dep)
	Log.Debug.Printf("  Container: %+ v", ci)
	Log.Debug.Printf("  Docker: %+ v", dockerInfo)

	depReq, err := dtos.LoadMap(&dtos.SingularityDeployRequest{}, sous.DTOMap{"Deploy": dep})
	if err != nil {
		return err
	}

	Log.Debug.Printf("Deploy req: %+ v", depReq)
	_, err = ra.singularityClient(cluster).Deploy(depReq.(*dtos.SingularityDeployRequest))
	return err
}

// PostRequest sends requests to Singularity to create a new Request
func (ra *RectiAgent) PostRequest(cluster, reqID string, instanceCount int) error {
	Log.Debug.Printf("Creating application %s %s %d", cluster, reqID, instanceCount)
	req, err := dtos.LoadMap(&dtos.SingularityRequest{}, sous.DTOMap{
		"Id":          reqID,
		"RequestType": dtos.SingularityRequestRequestTypeSERVICE,
		"Instances":   int32(instanceCount),
	})

	if err != nil {
		return err
	}

	Log.Debug.Printf("Create Request: %+ v", req)
	_, err = ra.singularityClient(cluster).PostRequest(req.(*dtos.SingularityRequest))
	return err
}

// DeleteRequest sends a request to Singularity to delete a request
func (ra *RectiAgent) DeleteRequest(cluster, reqID, message string) error {
	Log.Debug.Printf("Deleting application %s %s %s", cluster, reqID, message)
	req, err := dtos.LoadMap(&dtos.SingularityDeleteRequestRequest{}, sous.DTOMap{
		"Message": "Sous: " + message,
	})

	Log.Debug.Printf("Delete req: %+ v", req)
	_, err = ra.singularityClient(cluster).DeleteRequest(reqID,
		req.(*dtos.SingularityDeleteRequestRequest))
	return err
}

// Scale sends requests to Singularity to change the number of instances
// running for a given Request
func (ra *RectiAgent) Scale(cluster, reqID string, instanceCount int, message string) error {
	Log.Debug.Printf("Scaling %s %s %d %s", cluster, reqID, instanceCount, message)
	sr, err := dtos.LoadMap(&dtos.SingularityScaleRequest{}, sous.DTOMap{
		"ActionId": idify(uuid.NewV4().String()), // not positive this is appropriate
		// omitting DurationMillis - bears discussion
		"Instances":        int32(instanceCount),
		"Message":          "Sous" + message,
		"SkipHealthchecks": false,
	})

	Log.Debug.Printf("Scale req: %+ v", sr)
	_, err = ra.singularityClient(cluster).Scale(reqID, sr.(*dtos.SingularityScaleRequest))
	return err
}

// ImageName gets the container image name for a given deployment
func (ra *RectiAgent) ImageName(d *sous.Deployment) (string, error) {
	a, err := ra.nameCache.GetArtifact(d.SourceVersion)
	if err != nil {
		return "", err
	}
	return a.Name, nil
}

// ImageLabels gets the labels for an image name.
func (ra *RectiAgent) ImageLabels(in string) (map[string]string, error) {
	a := &sous.BuildArtifact{Name: in}
	sv, err := ra.nameCache.GetSourceVersion(a)
	if err != nil {
		return map[string]string{}, err
	}

	return docker.DockerLabels(sv), nil
}

func (ra *RectiAgent) getSingularityClient(url string) (*singularity.Client, bool) {
	ra.RLock()
	defer ra.RUnlock()
	cl, ok := ra.singClients[url]
	return cl, ok
}

func (ra *RectiAgent) singularityClient(url string) *singularity.Client {
	cl, ok := ra.getSingularityClient(url)
	if ok {
		return cl
	}
	ra.Lock()
	defer ra.Unlock()
	cl = singularity.NewClient(url)
	//cl.Debug = true
	ra.singClients[url] = cl
	return cl
}
