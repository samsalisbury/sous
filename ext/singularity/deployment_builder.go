package singularity

import (
	"fmt"

	singularity "github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/firsterr"
	"github.com/pkg/errors"
)

type (
	deploymentBuilder struct {
		//clusters     sous.Clusters
		//Deployment   sous.DeployState
		//imageName    string
		//depMarker    sDepMarker
		//deploy       sDeploy
		//failedDeploy failedDeploy
		//request      sRequest
		//req          Request
		//registry     sous.Registry
		Client *singularity.Client
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

	db.Deployment.Active.Cluster = &sous.Cluster{BaseURL: req.URL}
	db.request = req.RequestParent.Request

	return db.Deployment, db.canRetry(db.completeConstruction())
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
	db.Deployment.ActiveStatus = status
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

	failedDeployment, err := sousDeployment(deploy, db.request)
	if err != nil {
		return errors.Wrapf(err, "getting failed deployment (id: %q, request: %q)", deployID, requestID)
	}
	db.Deployment.Failed = failedDeployment
	db.Deployment.FailedReason = failureReason
	// TODO: Map deployment to sous.deployment.
	db.Deployment.FailedStatus = sous.DeployStatusFailed

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
	dh, err := db.retrieveDeployHistory(db.depMarker.RequestId, db.depMarker.DeployId)
	if err != nil {
		return err
	}
	db.deploy = dh.Deploy
	if db.deploy == nil {
		return malformedResponse{"Singularity deploy history included no deploy"}
	}
	return nil
}

// retrieveDeployHistory gets a single deploy history object, which contains
// the full singularity deploy object for a single deploy.
func (db *deploymentBuilder) retrieveDeployHistory(requestID, deployID string) (*dtos.SingularityDeployHistory, error) {
	sing := db.req.Client
	dh, err := sing.GetDeploy(requestID, deployID)
	if err != nil {
		Log.Debug.Printf("Failed to retrieve singularity deploy%q: %s", deployID, err)
		return nil, err
	}
	Log.Vomit.Printf("Retrived singularity deploy %q: %#v", deployID, dh)
	return dh, nil
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

	db.Deployment.Active.SourceID, err = docker.SourceIDFromLabels(labels)
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

		id := db.Deployment.ID()
		id.Cluster = nn

		checkID := MakeRequestID(id)
		sous.Log.Vomit.Printf("Trying hypothetical request ID: %s", checkID)
		if checkID == db.request.Id {
			db.Deployment.Active.ClusterName = nn
			sous.Log.Debug.Printf("Found cluster: %s", nn)
			break
		}
	}
	if db.Deployment.Active.ClusterName == "" {
		if matchCount == 1 {
			sous.Log.Debug.Printf("No request ID matched, using first plausible cluster: %s", posNick)
			db.Deployment.Active.ClusterName = posNick
			return nil
		}
		sous.Log.Debug.Printf("No cluster nickname (%#v) matched request id %s for %s", db.clusters, db.request.Id, db.imageName)
		return malformedResponse{fmt.Sprintf("No cluster nickname (%#v) matched request id %s", db.clusters, db.request.Id)}
	}

	return nil
}

// unpackDeployConfig maps the singularity data to a sous.Deployment (db.Target).
func (db *deploymentBuilder) unpackDeployConfig() error {
	d := &db.Deployment.Active
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
	mk, err := sousManifestKind(db.request.RequestType)
	if err != nil {
		return errors.Wrapf(err, "getting request type for %q", db.req.URL)
	}
	db.Deployment.Active.Kind = mk
	return nil
}

// mark: builder funcs

func sousManifestKind(t dtos.SingularityRequestRequestType) (sous.ManifestKind, error) {
	switch t {
	default:
		return "", fmt.Errorf("unknown request type %s", t)
	case dtos.SingularityRequestRequestTypeSERVICE:
		return sous.ManifestKindService, nil
	case dtos.SingularityRequestRequestTypeWORKER:
		return sous.ManifestKindWorker, nil
	case dtos.SingularityRequestRequestTypeON_DEMAND:
		return sous.ManifestKindOnDemand, nil
	case dtos.SingularityRequestRequestTypeSCHEDULED:
		return sous.ManifestKindScheduled, nil
	case dtos.SingularityRequestRequestTypeRUN_ONCE:
		return sous.ManifestKindOnce, nil
	}
}

func sousDeployConfig(sd *dtos.SingularityDeploy, sr *dtos.SingularityRequest) (*sous.DeployConfig, error) {
	d := &sous.DeployConfig{}
	d.Env = sd.Env
	Log.Vomit.Printf("Env: %+v", sd.Env)
	if d.Env == nil {
		d.Env = make(map[string]string)
	}

	singRez := sd.Resources
	if singRez == nil {
		return nil, malformedResponse{"Deploy object lacks resources field"}
	}
	d.Resources = make(sous.Resources)
	d.Resources["cpus"] = fmt.Sprintf("%f", singRez.Cpus)
	d.Resources["memory"] = fmt.Sprintf("%f", singRez.MemoryMb)
	d.Resources["ports"] = fmt.Sprintf("%d", singRez.NumPorts)

	d.NumInstances = int(sr.Instances)

	for _, v := range sd.ContainerInfo.Volumes {
		d.Volumes = append(d.Volumes,
			&sous.Volume{
				Host:      v.HostPath,
				Container: v.ContainerPath,
				Mode:      sous.VolumeMode(v.Mode),
			})
	}
	Log.Vomit.Printf("Volumes %+v", d.Volumes)
	if len(d.Volumes) > 0 {
		Log.Debug.Printf("%+v", d.Volumes[0])
	}

	return d, nil
}

func sousDeployment(sd *dtos.SingularityDeploy, sr *dtos.SingularityRequest) (*sous.Deployment, error) {
	d := &sous.Deployment{}
	d.Owners = make(sous.OwnerSet)
	for _, o := range sr.Owners {
		d.Owners.Add(o)
	}
	return d, nil
}
