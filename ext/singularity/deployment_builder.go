package singularity

import (
	"fmt"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/firsterr"
	"github.com/pkg/errors"
)

type (
	deploymentBuilder struct {
		clusters     sous.Clusters
		Target       sous.DeployState
		imageName    string
		depMarker    sDepMarker
		deploy       sDeploy
		failedDeploy failedDeploy
		request      sRequest
		req          Request
		registry     sous.Registry
	}

	failedDeploy struct {
		Reason string
		Deploy sDeploy
		Status sous.DeployStatus
	}

	canRetryRequest struct {
		cause error
		req   Request
	}

	malformedResponse struct {
		message string
	}
)

func (mr malformedResponse) Error() string {
	return mr.message
}

func isMalformed(err error) bool {
	err = errors.Cause(err)
	_, yes := err.(malformedResponse)
	Log.Vomit.Printf("err: %#v %T %t", err, err, yes)
	return yes
}

func (cr *canRetryRequest) Error() string {
	return fmt.Sprintf("%s: %s", cr.cause, cr.name())
}

func (cr *canRetryRequest) name() string {
	return fmt.Sprintf("%s:%s", cr.req.URL, cr.req.RequestParent.Request.Id)
}

func (db *deploymentBuilder) canRetry(err error) error {
	if err == nil || !db.isRetryable(err) {
		return err
	}
	return &canRetryRequest{err, db.req}
}

func (db *deploymentBuilder) isRetryable(err error) bool {
	return !isMalformed(err) &&
		db.req.URL != "" &&
		db.req.RequestParent != nil &&
		db.req.RequestParent.Request != nil &&
		db.req.RequestParent.Request.Id != ""
}

// BuildDeployment does all the work to collect the data for a Deployment
// from Singularity based on the initial SingularityRequest.
func BuildDeployment(reg sous.Registry, clusters sous.Clusters, req Request) (sous.DeployState, error) {
	Log.Vomit.Printf("%#v", req.RequestParent)
	db := deploymentBuilder{registry: reg, clusters: clusters, req: req}

	db.Target.Deployment.Cluster = &sous.Cluster{BaseURL: req.URL}
	db.request = req.RequestParent.Request

	return db.Target, db.canRetry(db.completeConstruction())
}

func (db *deploymentBuilder) completeConstruction() error {
	return firsterr.Returned(
		db.determineDeployStatus,
		db.retrieveDeploy,
		db.extractArtifactName,
		db.retrieveImageLabels,
		db.assignClusterName,
		db.unpackDeployConfig,
		db.determineManifestKind,
		db.determineFailedDeploy,
	)
}

func reqID(rp *dtos.SingularityRequestParent) (ID string) {
	defer func() {
		if e := recover(); e != nil {
			return
		}
	}()
	ID = "<null RP>"
	if rp != nil {
		ID = "<null Request>"
	}
	ID = rp.Request.Id
	return
}

// If there is a Pending deploy, as far as Sous is concerned, that's "to
// come" - we optimistically assume it will become Active, and that's the
// Deployment we should consider live.
//
// (At some point in the future we may want to be able to report the "live"
// deployment - at best based on this we could infer that a previous GDM
// entry was running. (consider several quick updates, though...(but
// Singularity semantics mean that each of them that was actually resolved
// would have been Active however briefly (but Sous would accept GDM updates
// arbitrarily quickly as compared to resolve completions...))))
//
// determineDeployStatus sets .Target.Status and .depMarker by calling
// determineDeployStatus.
func (db *deploymentBuilder) determineDeployStatus() error {
	logFDs("before retrieveDeploy")
	defer logFDs("after retrieveDeploy")

	rp := db.req.RequestParent
	if rp == nil {
		return malformedResponse{fmt.Sprintf("Singularity response didn't include a request parent. %v", db.req)}
	}

	if rp.RequestDeployState == nil {
		return malformedResponse{"Singularity response didn't include a deploy state. ReqId: " + reqID(rp)}
	}

	status, depMarker, err := determineDeployStatus(rp)
	if err != nil {
		return err
	}
	db.Target.Status = status
	db.depMarker = depMarker
	return nil
}

// determineDeployStatus tries to determine a sous.DeployStatus from the
// provided SingularityRequestParent, and also returns the related deploy
// marker. It does not take into account failed deploys.
func determineDeployStatus(rp *dtos.SingularityRequestParent) (sous.DeployStatus, *dtos.SingularityDeployMarker, error) {
	logFDs("before retrieveDeploy")
	defer logFDs("after retrieveDeploy")

	rds := rp.RequestDeployState
	if rds.PendingDeploy != nil {
		return sous.DeployStatusPending, rds.PendingDeploy, nil
	}
	if rds.ActiveDeploy != nil {
		return sous.DeployStatusActive, rds.ActiveDeploy, nil
	}
	return sous.DeployStatusUnknown, nil,
		malformedResponse{"Singularity deploy state included no dep markers. ReqID: " + reqID(rp)}
}

func (db *deploymentBuilder) determineFailedDeploy() error {

	// First, check if this deployment has already been attempted.
	//latestDeployResult = r.latestAttemptedDeploy(pair.ID(), computeRequestID(pair.Post))

	client := db.req.Client
	requestID := db.request.Id
	// Get latest deploy result.
	history, err := client.GetDeploys(requestID, 1, 1)
	if err != nil {
		return errors.Wrapf(err, "getting deploy history for request %q", requestID)
	}
	if len(history) == 0 {
		return nil // No history, thus no error.
	}
	latestDeployResult := history[0].DeployResult
	if latestDeployResult == nil {
		return nil // No details, thus no error.
	}
	if len(latestDeployResult.DeployFailures) == 0 {
		// Assuming that DeployFailures is always nonempty for failure states.
		return nil
	}
	deployID := latestDeployResult.DeployFailures[0].TaskId.DeployId
	failureReason := latestDeployResult.Message
	singleDeployHistory, err := client.GetDeploy(requestID, deployID)
	if err != nil {
		return errors.Wrapf(err, "getting deployment %q from request %q", deployID, requestID)
	}

	deploy := singleDeployHistory.Deploy

	db.failedDeploy = failedDeploy{
		Reason: failureReason,
		Deploy: deploy,
		// TODO: Map deploy statuses.
		Status: sous.DeployStatusFailed,
	}

	// TODO: Map deployment to sous.deployment.
	db.Target.FailedDeployment = &sous.Deployment{}
	db.Target.FailedDeploymentReason = failureReason

	return nil
}

//func sousDeployStatusFrom(s dtos.SingularityDeployResultDeployState) sous.DeployStatus {
//	switch s {
//	default:
//		return sous.DeployStatusUnknown
//		case dtos.SingularityDeployResultDeployStat
//	}
//}

func (db *deploymentBuilder) retrieveDeploy() error {
	// !!! makes HTTP req
	sing := db.req.Client
	dh, err := sing.GetDeploy(db.depMarker.RequestId, db.depMarker.DeployId)
	if err != nil {
		return err
	}
	Log.Vomit.Printf("%#v", dh)

	db.deploy = dh.Deploy
	if db.deploy == nil {
		return malformedResponse{"Singularity deploy history included no deploy"}
	}

	return nil
}

func (db *deploymentBuilder) extractArtifactName() error {
	logFDs("before retrieveImageLabels")
	defer logFDs("after retrieveImageLabels")
	ci := db.deploy.ContainerInfo
	if ci == nil {
		return malformedResponse{"Blank container info"}
	}

	if ci.Type != dtos.SingularityContainerInfoSingularityContainerTypeDOCKER {
		return malformedResponse{"Singularity container isn't a docker container"}
	}
	dkr := ci.Docker
	if dkr == nil {
		return malformedResponse{"Singularity deploy didn't include a docker info"}
	}

	db.imageName = dkr.Image
	return nil
}

func (db *deploymentBuilder) retrieveImageLabels() error {
	// XXX coupled to Docker registry as ImageMapper
	// !!! HTTP request
	labels, err := db.registry.ImageLabels(db.imageName)
	if err != nil {
		return malformedResponse{err.Error()}
	}
	Log.Vomit.Print("Labels: ", labels)

	db.Target.Deployment.SourceID, err = docker.SourceIDFromLabels(labels)
	if err != nil {
		return errors.Wrapf(malformedResponse{err.Error()}, "For reqID: %s", reqID(db.req.RequestParent))
	}

	return nil
}

func (db *deploymentBuilder) assignClusterName() error {
	var posNick string
	matchCount := 0
	for nn, url := range db.clusters {
		url := url.BaseURL
		if url != db.req.URL {
			continue
		}
		posNick = nn
		matchCount++

		id := db.Target.ID()
		id.Cluster = nn

		checkID := MakeRequestID(id)
		sous.Log.Vomit.Printf("Trying hypothetical request ID: %s", checkID)
		if checkID == db.request.Id {
			db.Target.Deployment.ClusterName = nn
			sous.Log.Debug.Printf("Found cluster: %s", nn)
			break
		}
	}
	if db.Target.Deployment.ClusterName == "" {
		if matchCount == 1 {
			sous.Log.Debug.Printf("No request ID matched, using first plausible cluster: %s", posNick)
			db.Target.Deployment.ClusterName = posNick
			return nil
		}
		sous.Log.Debug.Printf("No cluster nickname (%#v) matched request id %s for %s", db.clusters, db.request.Id, db.imageName)
		return malformedResponse{fmt.Sprintf("No cluster nickname (%#v) matched request id %s", db.clusters, db.request.Id)}
	}

	return nil
}

// unpackDeployConfig maps the singularity data to a sous.Deployment (db.Target).
func (db *deploymentBuilder) unpackDeployConfig() error {
	d := &db.Target.Deployment
	d.Env = db.deploy.Env
	Log.Vomit.Printf("Env: %+v", db.deploy.Env)
	if d.Env == nil {
		d.Env = make(map[string]string)
	}

	singRez := db.deploy.Resources
	if singRez == nil {
		return malformedResponse{"Deploy object lacks resources field"}
	}
	d.Resources = make(sous.Resources)
	d.Resources["cpus"] = fmt.Sprintf("%f", singRez.Cpus)
	d.Resources["memory"] = fmt.Sprintf("%f", singRez.MemoryMb)
	d.Resources["ports"] = fmt.Sprintf("%d", singRez.NumPorts)

	d.NumInstances = int(db.request.Instances)
	d.Owners = make(sous.OwnerSet)
	for _, o := range db.request.Owners {
		d.Owners.Add(o)
	}

	for _, v := range db.deploy.ContainerInfo.Volumes {
		d.DeployConfig.Volumes = append(d.DeployConfig.Volumes,
			&sous.Volume{
				Host:      v.HostPath,
				Container: v.ContainerPath,
				Mode:      sous.VolumeMode(v.Mode),
			})
	}
	Log.Vomit.Printf("Volumes %+v", d.DeployConfig.Volumes)
	if len(d.DeployConfig.Volumes) > 0 {
		Log.Debug.Printf("%+v", d.DeployConfig.Volumes[0])
	}

	return nil
}

func (db *deploymentBuilder) determineManifestKind() error {
	d := &db.Target.Deployment
	switch db.request.RequestType {
	default:
		return fmt.Errorf("Unrecognized response type returned by Singularity: %v", db.request.RequestType)
	case dtos.SingularityRequestRequestTypeSERVICE:
		d.Kind = sous.ManifestKindService
	case dtos.SingularityRequestRequestTypeWORKER:
		d.Kind = sous.ManifestKindWorker
	case dtos.SingularityRequestRequestTypeON_DEMAND:
		d.Kind = sous.ManifestKindOnDemand
	case dtos.SingularityRequestRequestTypeSCHEDULED:
		d.Kind = sous.ManifestKindScheduled
	case dtos.SingularityRequestRequestTypeRUN_ONCE:
		d.Kind = sous.ManifestKindOnce
	}
	return nil
}
