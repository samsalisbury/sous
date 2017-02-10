package singularity

import (
	"fmt"

	singularity "github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

type (
	deploymentBuilder struct {
		registry sous.Registry
		Original Request
		Client   *singularity.Client
		clusters sous.Clusters
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
)

func (cr *canRetryRequest) Error() string {
	return fmt.Sprintf("%s: %s", cr.cause, cr.name())
}

func (cr *canRetryRequest) name() string {
	return fmt.Sprintf("%s:%s", cr.req.URL, cr.req.RequestParent.Request.Id)
}

func (db *deploymentBuilder) canRetry(err error) error {
	if err == nil || !isRetryable(err) {
		return err
	}
	return &canRetryRequest{err, db.Original}
}

// DBError is a deploymentBuilder error.
type DBError struct {
	Req              Request
	IsMalformedError bool
	error
}

func (db *deploymentBuilder) MalformedResponsef(format string, a ...interface{}) DBError {
	return DBError{Req: db.Original, error: fmt.Errorf(format, a...), IsMalformedError: true}
}

// Errorf returns a formatted DBError.
func (db *deploymentBuilder) Errorf(format string, a ...interface{}) DBError {
	return DBError{Req: db.Original, error: fmt.Errorf(format, a...)}
}

// WrapErrorf returns a formatted error wrapping cause.
func (db *deploymentBuilder) WrapErrorf(cause error, format string, a ...interface{}) DBError {
	return DBError{Req: db.Original, error: errors.Wrapf(cause, format, a...)}
}

func isRetryable(err error) bool {
	dbErr, ok := err.(DBError)
	if !ok {
		// TODO: Check if the error implements net/Temporary?
		return true // we assume any other kind of error is retryable
	}
	if dbErr.IsMalformedError {
		Log.Debug.Printf("Received malformed response from Singularity: %s", err)
		return false
	}
	return dbErr.Req.URL != "" &&
		dbErr.Req.RequestParent != nil &&
		dbErr.Req.RequestParent.Request != nil &&
		dbErr.Req.RequestParent.Request.Id != ""
}

// BuildDeployment does all the work to collect the data for a Deployment
// from Singularity based on the initial SingularityRequest.
func BuildDeployment(reg sous.Registry, clusters sous.Clusters, req Request) (*sous.DeployState, error) {
	Log.Vomit.Printf("%#v", req.RequestParent)
	db := deploymentBuilder{registry: reg, clusters: clusters, Original: req}

	activeDeployment, err := db.sousDeployment(req.RequestParent.ActiveDeploy, req.RequestParent.Request)
	if err != nil {
		return nil, db.canRetry(err)
	}

	requestID := req.RequestParent.Request.Id

	failedDeployHistory, err := db.getFailedDeploy(requestID)
	if err != nil {
		return nil, db.canRetry(err)
	}

	failedDeployment, err := db.sousDeployment(failedDeployHistory.Deploy, req.RequestParent.Request)
	if err != nil {
		return nil, db.canRetry(err)
	}

	ds := &sous.DeployState{
		Active:       *activeDeployment,
		ActiveStatus: sous.DeployStatusActive, // TODO GET REAL STATUS
		Failed:       failedDeployment,
		FailedReason: failedDeployHistory.DeployResult.Message,
		FailedStatus: sous.DeployStatusFailed, // TODO GET REAL STATUS
	}

	ds.Active.Cluster = &sous.Cluster{BaseURL: req.URL}
	ds.Failed.Cluster = &sous.Cluster{BaseURL: req.URL}

	return ds, nil
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

// determineDeployStatus tries to determine a sous.DeployStatus from the
// provided SingularityRequestParent, and also returns the related deploy
// marker. It does not take into account failed deploys.
func (db *deploymentBuilder) determineDeployStatus(rp *dtos.SingularityRequestParent) (sous.DeployStatus, *dtos.SingularityDeployMarker, error) {
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
		db.MalformedResponsef("deploy state has no deploy markers for request %s", db.Original.URL)
}

func (db *deploymentBuilder) getFailedDeploy(requestID string) (*dtos.SingularityDeployHistory, error) {

	// First, check if this deployment has already been attempted.
	//latestDeployResult = r.latestAttemptedDeploy(pair.ID(), computeRequestID(pair.Post))

	// Get latest deploy result.
	history, err := db.Client.GetDeploys(requestID, 1, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "getting deploy history for request %q", requestID)
	}
	if len(history) == 0 {
		return nil, nil // No history, thus no error.
	}
	latestDeployResult := history[0].DeployResult
	if latestDeployResult == nil {
		return nil, nil // No details, thus no error.
	}
	if len(latestDeployResult.DeployFailures) == 0 {
		// Assuming that DeployFailures is always nonempty for failure states.
		return nil, nil
	}
	deployID := latestDeployResult.DeployFailures[0].TaskId.DeployId
	failureReason := latestDeployResult.Message
	singleDeployHistory, err := db.Client.GetDeploy(requestID, deployID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting failed deployment %q from request %q", deployID, requestID)
	}

	return singleDeployHistory, nil
}

// retrieveDeployHistory gets a single deploy history object, which contains
// the full singularity deploy object for a single deploy.
func (db *deploymentBuilder) retrieveDeployHistory(requestID, deployID string) (*dtos.SingularityDeployHistory, error) {
	dh, err := db.Client.GetDeploy(requestID, deployID)
	if err != nil {
		Log.Debug.Printf("Failed to retrieve singularity deploy%q: %s", deployID, err)
		return nil, err
	}
	Log.Vomit.Printf("Retrived singularity deploy %q: %#v", deployID, dh)
	return dh, nil
}

func (db *deploymentBuilder) getArtifact(deploy *dtos.SingularityDeploy) (*sous.BuildArtifact, error) {
	logFDs("before retrieveImageLabels")
	defer logFDs("after retrieveImageLabels")
	ci := deploy.ContainerInfo
	if ci == nil {
		return nil, db.MalformedResponsef("nil container info")
	}

	if ci.Type != dtos.SingularityContainerInfoSingularityContainerTypeDOCKER {
		return nil, db.MalformedResponsef("Singularity container isn't a docker container")
	}
	dkr := ci.Docker
	if dkr == nil {
		return nil, db.MalformedResponsef("Singularity deploy didn't include a docker info")
	}

	// TODO: Don't just assume docker here.
	// TODO: Add build qualities??
	return &sous.BuildArtifact{
		Name: dkr.Image,
		Type: "docker",
	}, nil
}

func (db *deploymentBuilder) retrieveSourceID(imageName string) (*sous.SourceID, error) {
	// XXX coupled to Docker registry as ImageMapper
	// !!! HTTP request
	labels, err := db.registry.ImageLabels(imageName)
	if err != nil {
		return nil, db.MalformedResponsef(err.Error())
	}
	Log.Vomit.Print("Labels: ", labels)

	sid, err := docker.SourceIDFromLabels(labels)
	if err != nil {
		return nil, db.MalformedResponsef(err.Error())
	}

	return &sid, nil
}

func (db *deploymentBuilder) getClusterName(did sous.DeployID, sr *dtos.SingularityRequest) (string, error) {
	var clusterName string
	matchCount := 0
	for nn, cluster := range db.clusters {
		if cluster.BaseURL != db.Original.URL {
			continue
		}
		clusterName = nn
		matchCount++
		did.Cluster = nn
		checkID := MakeRequestID(did)
		sous.Log.Vomit.Printf("Trying hypothetical request ID: %s", checkID)
		if checkID == sr.Id {
			clusterName = nn
			sous.Log.Debug.Printf("Found cluster: %s", nn)
			break
		}
	}
	if clusterName == "" {
		if matchCount == 1 {
			sous.Log.Debug.Printf("No request ID matched, using first plausible cluster: %s", clusterName)
			db.Deployment.Active.ClusterName = clusterName
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

func (db *deploymentBuilder) sousDeployment(sd *dtos.SingularityDeploy, sr *dtos.SingularityRequest) (*sous.Deployment, error) {
	d := &sous.Deployment{}
	d.Owners = make(sous.OwnerSet)
	for _, o := range sr.Owners {
		d.Owners.Add(o)
	}
	return d, nil
}
