package singularity

import (
	"fmt"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
)

type (
	deploymentBuilder struct {
		Target        sous.Deployment
		depMarker     sDepMarker
		deploy        sDeploy
		request       sRequest
		req           SingReq
		rectification rectificationClient
	}

	canRetryRequest struct {
		cause error
		req   SingReq
	}

	malformedResponse struct {
		message string
	}
)

func (mr malformedResponse) Error() string {
	return mr.message
}

func (cr *canRetryRequest) Error() string {
	return fmt.Sprintf("%s: %s", cr.cause, cr.name())
}

func (cr *canRetryRequest) name() string {
	return fmt.Sprintf("%s:%s", cr.req.SourceURL, cr.req.ReqParent.Request.Id)
}

// NewDeploymentBuilder creates a deploymentBuilder prepared to collect the
// data associated with req and return a Deployment
func NewDeploymentBuilder(cl rectificationClient, req SingReq) deploymentBuilder {
	return deploymentBuilder{rectification: cl, req: req}
}

func (uc *deploymentBuilder) canRetry(err error) error {
	if _, ok := err.(malformedResponse); ok {
		return err
	}

	if uc.req.SourceURL == "" {
		return err
	}

	if uc.req.ReqParent == nil {
		return err
	}
	if uc.req.ReqParent.Request == nil {
		return err
	}

	if uc.req.ReqParent.Request.Id == "" {
		return err
	}

	return &canRetryRequest{err, uc.req}
}

// TODO: Unexport this method.
func (uc *deploymentBuilder) CompleteConstruction() error {
	uc.Target.Cluster = uc.req.SourceURL
	uc.request = uc.req.ReqParent.Request

	err := uc.retrieveDeploy()
	if err != nil {
		return uc.canRetry(err)
	}

	err = uc.retrieveImageLabels()
	if err != nil {
		return uc.canRetry(err)
	}

	err = uc.unpackDeployConfig()
	if err != nil {
		return uc.canRetry(err)
	}

	err = uc.determineManifestKind()
	if err != nil {
		return uc.canRetry(err)
	}

	return nil
}

func (uc *deploymentBuilder) retrieveDeploy() error {

	rp := uc.req.ReqParent
	rds := rp.RequestDeployState
	sing := uc.req.Sing

	if rds == nil {
		return malformedResponse{"Singularity response didn't include a deploy state. ReqId: " + rp.Request.Id}
	}
	uc.depMarker = rds.PendingDeploy
	if uc.depMarker == nil {
		uc.depMarker = rds.ActiveDeploy
	}
	if uc.depMarker == nil {
		return malformedResponse{"Singularity deploy state included no dep markers. ReqID: " + rp.Request.Id}
	}

	// !!! makes HTTP req
	dh, err := sing.GetDeploy(uc.depMarker.RequestId, uc.depMarker.DeployId)
	if err != nil {
		return err
	}

	uc.deploy = dh.Deploy
	if uc.deploy == nil {
		return malformedResponse{"Singularity deploy history included no deploy"}
	}

	return nil
}

func (uc *deploymentBuilder) retrieveImageLabels() error {
	ci := uc.deploy.ContainerInfo
	if ci.Type != dtos.SingularityContainerInfoSingularityContainerTypeDOCKER {
		return fmt.Errorf("Singularity container isn't a docker container")
	}
	dkr := ci.Docker
	if dkr == nil {
		return malformedResponse{"Singularity deploy didn't include a docker info"}
	}

	imageName := dkr.Image

	// !!! HTTP request
	labels, err := uc.rectification.ImageLabels(imageName)
	if err != nil {
		return malformedResponse{err.Error()}
	}
	Log.Vomit.Print("Labels: ", labels)

	uc.Target.SourceVersion, err = docker.SourceVersionFromLabels(labels)
	if err != nil {
		return malformedResponse{fmt.Sprintf("For reqID: %s, %s", uc.req.ReqParent.Request.Id, err.Error())}
	}

	return nil
}

func (uc *deploymentBuilder) unpackDeployConfig() error {
	uc.Target.Env = uc.deploy.Env
	Log.Vomit.Printf("Env %+v", uc.deploy.Env)
	if uc.Target.Env == nil {
		uc.Target.Env = make(map[string]string)
	}

	singRez := uc.deploy.Resources
	uc.Target.Resources = make(sous.Resources)
	uc.Target.Resources["cpus"] = fmt.Sprintf("%f", singRez.Cpus)
	uc.Target.Resources["memory"] = fmt.Sprintf("%f", singRez.MemoryMb)
	uc.Target.Resources["ports"] = fmt.Sprintf("%d", singRez.NumPorts)

	uc.Target.NumInstances = int(uc.request.Instances)
	uc.Target.Owners = make(sous.OwnerSet)
	for _, o := range uc.request.Owners {
		uc.Target.Owners.Add(o)
	}

	for _, v := range uc.deploy.ContainerInfo.Volumes {
		uc.Target.DeployConfig.Volumes = append(uc.Target.DeployConfig.Volumes,
			&sous.Volume{
				Host:      v.HostPath,
				Container: v.ContainerPath,
				Mode:      sous.VolumeMode(v.Mode),
			})
	}
	Log.Vomit.Printf("Volumes %+v", uc.Target.DeployConfig.Volumes)
	if len(uc.Target.DeployConfig.Volumes) > 0 {
		Log.Debug.Printf("%+v", uc.Target.DeployConfig.Volumes[0])
	}

	return nil
}

func (uc *deploymentBuilder) determineManifestKind() error {
	switch uc.request.RequestType {
	default:
		return fmt.Errorf("Unrecognized response type returned by Singularity: %v", uc.request.RequestType)
	case dtos.SingularityRequestRequestTypeSERVICE:
		uc.Target.Kind = sous.ManifestKindService
	case dtos.SingularityRequestRequestTypeWORKER:
		uc.Target.Kind = sous.ManifestKindWorker
	case dtos.SingularityRequestRequestTypeON_DEMAND:
		uc.Target.Kind = sous.ManifestKindOnDemand
	case dtos.SingularityRequestRequestTypeSCHEDULED:
		uc.Target.Kind = sous.ManifestKindScheduled
	case dtos.SingularityRequestRequestTypeRUN_ONCE:
		uc.Target.Kind = sous.ManifestKindOnce
	}
	return nil
}
