package sous

import (
	"fmt"

	"github.com/opentable/singularity/dtos"
	"github.com/opentable/sous/util/docker_registry"
)

type (
	deploymentBuilder struct {
		target         Deployment
		depMarker      sDepMarker
		deploy         sDeploy
		request        sRequest
		req            singReq
		registryClient docker_registry.Client
	}

	canRetryRequest struct {
		cause error
		req   singReq
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
	return fmt.Sprintf("%s:%s", cr.req.sourceURL, cr.req.reqParent.Request.Id)
}

func newDeploymentBuilder(cl docker_registry.Client, req singReq) deploymentBuilder {
	return deploymentBuilder{registryClient: cl, req: req}
}

func (uc *deploymentBuilder) canRetry(err error) error {
	if _, ok := err.(malformedResponse); ok {
		return err
	}

	if uc.req.sourceURL == "" {
		return err
	}

	if uc.req.reqParent == nil {
		return err
	}
	if uc.req.reqParent.Request == nil {
		return err
	}

	if uc.req.reqParent.Request.Id == "" {
		return err
	}

	return &canRetryRequest{err, uc.req}
}

func (uc *deploymentBuilder) completeConstruction() error {
	uc.target.Cluster = uc.req.sourceURL
	uc.request = uc.req.reqParent.Request

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

	rp := uc.req.reqParent
	rds := rp.RequestDeployState
	sing := uc.req.sing

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
	labels, err := uc.registryClient.LabelsForImageName(imageName)
	if err != nil {
		return err
	}

	uc.target.SourceVersion, err = SourceVersionFromLabels(labels)
	if err != nil {
		return malformedResponse{fmt.Sprintf("For reqID: %s, %s", uc.req.reqParent.Request.Id, err.Error())}
	}

	return nil
}

func (uc *deploymentBuilder) unpackDeployConfig() error {
	uc.target.Env = uc.deploy.Env
	if uc.target.Env == nil {
		uc.target.Env = make(map[string]string)
	}

	singRez := uc.deploy.Resources
	uc.target.Resources = make(Resources)
	uc.target.Resources["cpus"] = fmt.Sprintf("%f", singRez.Cpus)
	uc.target.Resources["memory"] = fmt.Sprintf("%f", singRez.MemoryMb)
	uc.target.Resources["ports"] = fmt.Sprintf("%d", singRez.NumPorts)

	uc.target.NumInstances = int(uc.request.Instances)
	for _, o := range uc.request.Owners {
		uc.target.Owners.Add(o)
	}

	return nil
}

func (uc *deploymentBuilder) determineManifestKind() error {
	switch uc.request.RequestType {
	default:
		return fmt.Errorf("Unrecognized response type returned by Singularity: %v", uc.request.RequestType)
	case dtos.SingularityRequestRequestTypeSERVICE:
		uc.target.Kind = ManifestKindService
	case dtos.SingularityRequestRequestTypeWORKER:
		uc.target.Kind = ManifestKindWorker
	case dtos.SingularityRequestRequestTypeON_DEMAND:
		uc.target.Kind = ManifestKindOnDemand
	case dtos.SingularityRequestRequestTypeSCHEDULED:
		uc.target.Kind = ManifestKindScheduled
	case dtos.SingularityRequestRequestTypeRUN_ONCE:
		uc.target.Kind = ManifestKindOnce
	}
	return nil
}
