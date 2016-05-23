package sous

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"

	"github.com/opentable/singularity"
	"github.com/opentable/singularity/dtos"
	"github.com/satori/go.uuid"
)

/*
The imagined use case here is like this:

intendedSet := getFromManifests()
existingSet := getFromSingularity()

dChans := intendedSet.Diff(existingSet)

Rectify(dChans)
*/

type (
	rectifier struct {
		sing RectificationClient
	}

	// RectificationClient abstracts the raw interactions with Singularity.
	// The methods on this interface are tightly bound to the semantics of Singularity itself -
	// it's recommended to interact with the Sous Recify function or the recitification driver
	// rather than with implentations of this interface directly.
	RectificationClient interface {
		// Deploy creates a new deploy on a particular requeust
		Deploy(cluster, depID, reqId, dockerImage string, r Resources) error

		// PostRequest sends a request to a Singularity cluster to initiate
		PostRequest(cluster, reqID string, instanceCount int) error

		//Scale updates the instanceCount associated with a request
		Scale(cluster, reqID string, instanceCount int, message string) error

		//ImageName finds or guesses a docker image name for a Deployment
		ImageName(d *Deployment) (string, error)
	}

	RectiAgent struct {
		singClients map[string]*singularity.Client
		nameCache   NameCache
	}

	dtoMap map[string]interface{}

	CreateError struct {
		Deployment *Deployment
		Err        error
	}

	DeleteError struct {
		Deployment *Deployment
		Err        error
	}

	ChangeError struct {
		Deployments DeploymentPair
		Err         error
	}

	RectificationError interface {
		error
		ExistingDeployment() *Deployment
		IntendedDeployment() *Deployment
	}
)

func (e *CreateError) Error() string {
	return fmt.Sprintf("Couldn't create deployment %+v: %v", e.Deployment, e.Err)
}

func (e *CreateError) ExistingDeployment() *Deployment {
	return nil
}

func (e *CreateError) IntendedDeployment() *Deployment {
	return e.Deployment
}

func (e *DeleteError) Error() string {
	return fmt.Sprintf("Couldn't delete deployment %+v: %v", e.Deployment, e.Err)
}

func (e *DeleteError) ExistingDeployment() *Deployment {
	return e.Deployment
}

func (e *DeleteError) IntendedDeployment() *Deployment {
	return nil
}

func (e *ChangeError) Error() string {
	return fmt.Sprintf("Couldn't change from deployment %+v to deployment %+v: %v", e.Deployments.prior, e.Deployments.post, e.Err)
}

func (e *ChangeError) ExistingDeployment() *Deployment {
	return e.Deployments.prior
}

func (e *ChangeError) IntendedDeployment() *Deployment {
	return e.Deployments.post
}

func Rectify(dcs DiffChans, errs chan<- RectificationError, s RectificationClient) {
	rect := rectifier{s}
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() { rect.rectifyCreates(dcs.Created, errs); wg.Done() }()
	go func() { rect.rectifyDeletes(dcs.Deleted, errs); wg.Done() }()
	go func() { rect.rectifyModifys(dcs.Modified, errs); wg.Done() }()
	go func() { wg.Wait(); close(errs) }()
}

func (ra *RectiAgent) Deploy(cluster, depID, reqId, dockerImage string, r Resources) error {
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
		"RequestId":     reqId,
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

func (ra *RectiAgent) Scale(cluster, reqID string, instanceCount int, message string) error {
	sr, err := dtos.LoadMap(&dtos.SingularityScaleRequest{}, dtoMap{
		"ActionId": idify(uuid.NewV4().String()), // not positive this is appropriate
		// omitting DurationMillis - bears discussion
		"Instances":        int32(instanceCount),
		"Message":          message,
		"SkipHealthchecks": false,
	})

	_, err = ra.singularityClient(cluster).Scale(reqID, sr.(*dtos.SingularityScaleRequest))
	return err
}

func (ra *RectiAgent) ImageName(d *Deployment) (string, error) {
	return ra.nameCache.GetImageName(d.SourceVersion)
}

func (ra *RectiAgent) singularityClient(url string) *singularity.Client {
	if cl, ok := ra.singClients[url]; ok {
		return cl
	} else {
		cl = singularity.NewClient(url)
		ra.singClients[url] = cl
		return cl
	}
}

func (r *rectifier) rectifyCreates(cc chan Deployment, errs chan<- RectificationError) {
	for d := range cc {
		name, err := r.sing.ImageName(&d)
		if err != nil {
			errs <- &CreateError{Deployment: &d, Err: err}
			continue
		}

		reqID := computeRequestId(&d)
		err = r.sing.PostRequest(d.Cluster, reqID, d.NumInstances)
		if err != nil {
			errs <- &CreateError{Deployment: &d, Err: err}
			continue
		}

		err = r.sing.Deploy(d.Cluster, newDepID(), reqID, name, d.Resources)
		if err != nil {
			errs <- &CreateError{Deployment: &d, Err: err}
			continue
		}
	}
}

func (r *rectifier) rectifyDeletes(dc chan Deployment, errs chan<- RectificationError) {
	for d := range dc {
		err := r.sing.Scale(d.Cluster, computeRequestId(&d), 0, "scaling deleted manifest to zero")
		if err != nil {
			errs <- &DeleteError{Deployment: &d, Err: err}
			continue
		}
	}
}

func (r *rectifier) rectifyModifys(mc chan DeploymentPair, errs chan<- RectificationError) {
	for pair := range mc {
		if r.changesReq(pair) {
			err := r.sing.Scale(pair.post.Cluster, computeRequestId(pair.post), pair.post.NumInstances, "rectified scaling")
			if err != nil {
				errs <- &ChangeError{Deployments: pair, Err: err}
				continue
			}
		}

		if changesDep(pair) {
			name, err := r.sing.ImageName(pair.post)
			if err != nil {
				errs <- &ChangeError{Deployments: pair, Err: err}
				continue
			}

			err = r.sing.Deploy(pair.post.Cluster, newDepID(), computeRequestId(pair.prior), name, pair.post.Resources)
			if err != nil {
				errs <- &ChangeError{Deployments: pair, Err: err}
				continue
			}
		}
	}
}

func (r rectifier) changesReq(pair DeploymentPair) bool {
	return pair.prior.NumInstances != pair.post.NumInstances
}

func changesDep(pair DeploymentPair) bool {
	return !(pair.prior.SourceVersion.Equal(pair.post.SourceVersion) && pair.prior.Resources.Equal(pair.prior.Resources))
}

func computeRequestId(d *Deployment) string {
	if len(d.RequestId) > 0 {
		return d.RequestId
	}
	return d.SourceVersion.CanonicalName().String()
}

var notInIdRE = regexp.MustCompile(`[-/]`)

func idify(in string) string {
	return notInIdRE.ReplaceAllString(in, "")
}

func newDepID() string {
	return idify(uuid.NewV4().String())
}

func BuildSingRequest(reqID string, instances int) *dtos.SingularityRequest {
	req := dtos.SingularityRequest{}
	req.LoadMap(map[string]interface{}{
		"Id":          reqID,
		"RequestType": dtos.SingularityRequestRequestTypeSERVICE,
		"Instances":   int32(instances),
	})
	return &req
}

func BuildSingDeployRequest(depID, reqID, imageName string, res Resources) *dtos.SingularityDeployRequest {
	resCpuS, ok := res["cpus"]
	if !ok {
		return nil
	}

	// Ugh. Double blinding of the types for this...
	resCpu, err := strconv.ParseFloat(resCpuS, 64)
	if err != nil {
		return nil
	}

	resMemS, ok := res["memoryMb"]
	if !ok {
		return nil
	}

	resMem, err := strconv.ParseFloat(resMemS, 64)
	if err != nil {
		return nil
	}

	resPortsS, ok := res["numPorts"]
	if !ok {
		return nil
	}

	resPorts, err := strconv.ParseInt(resPortsS, 10, 32)
	if err != nil {
		return nil
	}

	di := dtos.SingularityDockerInfo{}
	di.LoadMap(map[string]interface{}{
		"Image": imageName,
	})

	rez := dtos.Resources{}
	rez.LoadMap(map[string]interface{}{
		"Cpus":     resCpu,
		"MemoryMb": resMem,
		"NumPorts": resPorts,
	})

	ci := dtos.SingularityContainerInfo{}
	ci.LoadMap(map[string]interface{}{
		"Type":   dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
		"Docker": di,
	})

	dep := dtos.SingularityDeploy{}
	dep.LoadMap(map[string]interface{}{
		"Id":            depID,
		"RequestId":     reqID,
		"Resources":     rez,
		"ContainerInfo": ci,
	})

	dr := dtos.SingularityDeployRequest{}
	dr.LoadMap(map[string]interface{}{
		"Deploy": &dep,
	})

	return &dr
}

func BuildScaleRequest(num int, message string) *dtos.SingularityScaleRequest {
	sr := dtos.SingularityScaleRequest{}
	sr.LoadMap(map[string]interface{}{
		"Id":             newDepID(),
		"Instances":      int32(num),
		"Message":        message,
		"DurationMillis": 60000, // N.b. yo creo this is how long Singularity will allow this attempt to take.
	})
	return &sr
}
