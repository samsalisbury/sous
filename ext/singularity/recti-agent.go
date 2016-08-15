package singularity

import (
	"regexp"
	"sync"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/swaggering"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

var illegalDeployIDChars = regexp.MustCompile(`[-/:]`)

// MakeDeployID cleans a string to be used as a Singularity deploy ID.
func MakeDeployID(in string) string {
	return illegalDeployIDChars.ReplaceAllString(in, "")
}

// RectiAgent is an implementation of the RectificationClient interface
type RectiAgent struct {
	singClients map[string]*singularity.Client
	sync.RWMutex
	nameCache sous.Registry
}

// NewRectiAgent returns a set-up RectiAgent
func NewRectiAgent(b sous.Registry) *RectiAgent {
	return &RectiAgent{
		singClients: make(map[string]*singularity.Client),
		nameCache:   b,
	}
}

// SingMap produces a DTOMap appropriate for building a Singularity
// dto.Resources struct from
func MapResources(r sous.Resources) dtoMap {
	return dtoMap{
		"Cpus":     r.Cpus(),
		"MemoryMb": r.Memory(),
		"NumPorts": int32(r.Ports()),
	}
}

// Deploy sends requests to Singularity to make a deployment happen
func (ra *RectiAgent) Deploy(cluster, depID, reqID, dockerImage string,
	r sous.Resources, e sous.Env, vols sous.Volumes) error {
	Log.Debug.Printf("Deploying instance %s %s %s %s %v %v", cluster, depID, reqID, dockerImage, r, e)
	depReq, err := buildDeployRequest(dockerImage, e, r, reqID, vols)
	if err != nil {
		return err
	}

	Log.Debug.Printf("Deploy req: %+ v", depReq)
	_, err = ra.singularityClient(cluster).Deploy(depReq)
	return err
}

func buildDeployRequest(dockerImage string, e sous.Env, r sous.Resources, reqID string, vols sous.Volumes) (*dtos.SingularityDeployRequest, error) {
	var depReq swaggering.Fielder
	dockerInfo, err := swaggering.LoadMap(&dtos.SingularityDockerInfo{}, dtoMap{
		"Image":   dockerImage,
		"Network": dtos.SingularityDockerInfoSingularityDockerNetworkTypeBRIDGE, //defaulting to all bridge
	})
	if err != nil {
		return nil, err
	}

	res, err := swaggering.LoadMap(&dtos.Resources{}, MapResources(r))
	if err != nil {
		return nil, err
	}

	vs := dtos.SingularityVolumeList{}
	for _, v := range vols {
		sv, err := swaggering.LoadMap(&dtos.SingularityVolume{}, dtoMap{
			"ContainerPath": v.Container,
			"HostPath":      v.Host,
			"Mode":          dtos.SingularityVolumeSingularityDockerVolumeMode(string(v.Mode)),
		})
		if err != nil {
			return nil, err
		}
		vs = append(vs, sv.(*dtos.SingularityVolume))
	}

	ci, err := swaggering.LoadMap(&dtos.SingularityContainerInfo{}, dtoMap{
		"Type":    dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
		"Docker":  dockerInfo,
		"Volumes": vs,
	})
	if err != nil {
		return nil, err
	}

	dep, err := swaggering.LoadMap(&dtos.SingularityDeploy{}, dtoMap{
		"Id":            MakeDeployID(uuid.NewV4().String()),
		"RequestId":     reqID,
		"Resources":     res,
		"ContainerInfo": ci,
		"Env":           map[string]string(e),
	})
	Log.Debug.Printf("Deploy: %+ v", dep)
	Log.Debug.Printf("  Container: %+ v", ci)
	Log.Debug.Printf("  Docker: %+ v", dockerInfo)

	depReq, err = swaggering.LoadMap(&dtos.SingularityDeployRequest{}, dtoMap{"Deploy": dep})
	if err != nil {
		return nil, err
	}
	return depReq.(*dtos.SingularityDeployRequest), nil
}

// PostRequest sends requests to Singularity to create a new Request
func (ra *RectiAgent) PostRequest(cluster, reqID string, instanceCount int) error {
	Log.Debug.Printf("Creating application %s %s %d", cluster, reqID, instanceCount)
	req, err := swaggering.LoadMap(&dtos.SingularityRequest{}, dtoMap{
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
	req, err := swaggering.LoadMap(&dtos.SingularityDeleteRequestRequest{}, dtoMap{
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
	sr, err := swaggering.LoadMap(&dtos.SingularityScaleRequest{}, dtoMap{
		"ActionId": MakeDeployID(uuid.NewV4().String()), // not positive this is appropriate
		// omitting DurationMillis - bears discussion
		"Instances":        int32(instanceCount),
		"Message":          "Sous" + message,
		"SkipHealthchecks": false,
	})

	Log.Debug.Printf("Scale req: %+ v", sr)
	_, err = ra.singularityClient(cluster).Scale(reqID, sr.(*dtos.SingularityScaleRequest))
	return err
}

// ImageLabels gets the labels for an image name.
func (ra *RectiAgent) ImageLabels(in string) (map[string]string, error) {
	a := docker.NewBuildArtifact(in)
	sv, err := ra.nameCache.GetSourceID(a)
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "Image name: %s", in)
	}

	return docker.Labels(sv), nil
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
