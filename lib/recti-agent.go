package sous

import (
	"sync"

	"github.com/opentable/singularity"
	"github.com/opentable/singularity/dtos"
	"github.com/satori/go.uuid"
)

// RectiAgent is an implementation of the RectificationClient interface
type RectiAgent struct {
	singClients map[string]*singularity.Client
	sync.RWMutex
	nameCache ImageMapper
}

// NewRectiAgent returns a set-up RectiAgent
func NewRectiAgent(nc ImageMapper) *RectiAgent {
	return &RectiAgent{
		singClients: make(map[string]*singularity.Client),
		nameCache:   nc,
	}
}

// Deploy sends requests to Singularity to make a deployment happen
func (ra *RectiAgent) Deploy(cluster, depID, reqID, dockerImage string, r Resources) error {
	dockerInfo, err := dtos.LoadMap(&dtos.SingularityDockerInfo{}, dtoMap{
		"Image": dockerImage,
	})
	if err != nil {
		return err
	}

	res, err := dtos.LoadMap(&dtos.Resources{}, dtoMap{
		"Cpus":     0.1,
		"MemoryMb": 100.0,
		"NumPorts": int32(1),
	})
	if err != nil {
		return err
	}

	ci, err := dtos.LoadMap(&dtos.SingularityContainerInfo{}, dtoMap{
		"Type":   dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
		"Docker": dockerInfo,
	})
	if err != nil {
		return err
	}

	dep, err := dtos.LoadMap(&dtos.SingularityDeploy{}, dtoMap{
		"Id":            idify(uuid.NewV4().String()),
		"RequestId":     reqID,
		"Resources":     res,
		"ContainerInfo": ci,
	})

	depReq, err := dtos.LoadMap(&dtos.SingularityDeployRequest{}, dtoMap{"Deploy": dep})
	if err != nil {
		return err
	}

	_, err = ra.singularityClient(cluster).Deploy(depReq.(*dtos.SingularityDeployRequest))
	return err
}

// PostRequest sends requests to Singularity to create a new Request
func (ra *RectiAgent) PostRequest(cluster, reqID string, instanceCount int) error {
	req, err := dtos.LoadMap(&dtos.SingularityRequest{}, dtoMap{
		"Id":          reqID,
		"RequestType": dtos.SingularityRequestRequestTypeSERVICE,
		"Instances":   int32(instanceCount),
	})

	if err != nil {
		return err
	}

	_, err = ra.singularityClient(cluster).PostRequest(req.(*dtos.SingularityRequest))
	return err
}

// DeleteRequest sends a request to Singularity to delete a request
func (ra *RectiAgent) DeleteRequest(cluster, reqID, message string) error {
	req, err := dtos.LoadMap(&dtos.SingularityDeleteRequestRequest{}, dtoMap{
		"Message": "Sous: " + message,
	})
	_, err = ra.singularityClient(cluster).DeleteRequest(reqID, req.(*dtos.SingularityDeleteRequestRequest))
	return err
}

// Scale sends requests to Singularity to change the number of instances
// running for a given Request
func (ra *RectiAgent) Scale(cluster, reqID string, instanceCount int, message string) error {
	sr, err := dtos.LoadMap(&dtos.SingularityScaleRequest{}, dtoMap{
		"ActionId": idify(uuid.NewV4().String()), // not positive this is appropriate
		// omitting DurationMillis - bears discussion
		"Instances":        int32(instanceCount),
		"Message":          "Sous" + message,
		"SkipHealthchecks": false,
	})

	_, err = ra.singularityClient(cluster).Scale(reqID, sr.(*dtos.SingularityScaleRequest))
	return err
}

// ImageName gets the container image name for a given deployment
func (ra *RectiAgent) ImageName(d *Deployment) (string, error) {
	return ra.nameCache.GetImageName(d.SourceVersion)
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
	ra.singClients[url] = cl
	return cl
}
