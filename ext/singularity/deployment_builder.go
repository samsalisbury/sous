package singularity

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/firsterr"
	"github.com/pkg/errors"
)

type (
	deploymentBuilder struct {
		clusters  sous.Clusters
		Target    sous.DeployState
		imageName string
		depMarker sDepMarker
		history   sHistory
		deploy    sDeploy
		request   sRequest
		req       SingReq
		registry  sous.ImageLabeller
		reqID     string
	}

	canRetryRequest struct {
		cause error
		req   SingReq
	}

	malformedResponse struct {
		message string
	}

	nonSousError struct {
	}

	notThisClusterError struct {
		foundClusterName        string
		responsibleClusterNames []string
	}
)

func (ntc notThisClusterError) Error() string {
	return fmt.Sprintf("%s does not belong to this Sous server %#v.",
		ntc.foundClusterName, ntc.responsibleClusterNames)
}

func (nsd nonSousError) Error() string {
	return "Not a Sous SingularityDeploy."
}

func ignorableDeploy(err error) bool {
	switch errors.Cause(err).(type) {
	case nonSousError, notThisClusterError:
		return true
	}
	return false
}

func (mr malformedResponse) Error() string {
	return mr.message
}

func isMalformed(err error) bool {
	err = errors.Cause(err)
	_, isMal := err.(malformedResponse)
	Log.Vomit.Printf("is malformedResponse? err: %+v %T %t", err, err, isMal)
	_, isUMT := err.(*json.UnmarshalTypeError)
	Log.Vomit.Printf("is json unmarshal type error? err: %+v %T %t", err, err, isUMT)
	_, isUMF := err.(*json.UnmarshalFieldError)
	Log.Vomit.Printf("is json unmarshal value error? err: %+v %T %t", err, err, isUMF)
	_, isUST := err.(*json.UnsupportedTypeError)
	Log.Vomit.Printf("is json unsupported type error? err: %+v %T %t", err, err, isUST)
	_, isUSV := err.(*json.UnsupportedValueError)
	Log.Vomit.Printf("is json unsupported value error? err: %+v %T %t", err, err, isUSV)
	return isMal || isUMT || isUMF || isUST || isUSV
}

func (cr *canRetryRequest) Error() string {
	return fmt.Sprintf("%s: %s", cr.cause, cr.name())
}

func (cr *canRetryRequest) name() string {
	return fmt.Sprintf("%s:%s", cr.req.SourceURL, cr.req.ReqParent.Request.Id)
}

func (db *deploymentBuilder) canRetry(err error) error {
	if err == nil || !db.isRetryable(err) {
		return err
	}
	return &canRetryRequest{err, db.req}
}

func (db *deploymentBuilder) isRetryable(err error) bool {
	return !isMalformed(err) &&
		!ignorableDeploy(err) &&
		db.req.SourceURL != "" &&

		db.req.ReqParent != nil &&
		db.req.ReqParent.Request != nil &&
		db.req.ReqParent.Request.Id != ""
}

// BuildDeployment does all the work to collect the data for a Deployment
// from Singularity based on the initial SingularityRequest.
func BuildDeployment(reg sous.ImageLabeller, clusters sous.Clusters, req SingReq) (sous.DeployState, error) {
	Log.Vomit.Printf("%#v", req.ReqParent)
	db := deploymentBuilder{registry: reg, clusters: clusters, req: req}
	return db.Target, db.canRetry(db.completeConstruction())
}

func (db *deploymentBuilder) completeConstruction() error {
	wrapError := func(fn func() error, msgStr string) func() error {
		return func() error {
			return errors.Wrap(fn(), msgStr)
		}
	}
	return firsterr.Returned(
		wrapError(db.basics, "Failed to extract basic information from original request."),
		wrapError(db.determineDeployStatus, "Failed to determine deploy status."),
		wrapError(db.retrieveDeployHistory, "Failed to retrieve SingularityDeployHistory from SingularityRequestParent."),
		wrapError(db.extractDeployFromDeployHistory, "Failed to extract SingularityDeploy from SingularityDeployHistory."),
		wrapError(db.sousDeployCheck, "Could not determine if the SingularityDeploy is controlled by Sous"),
		wrapError(db.determineStatus, "Could not determine current status of SingularityDeploy"),
		wrapError(db.extractArtifactName, "Could not extract ArtifactName (Docker image name) from SingularityDeploy."),
		wrapError(db.retrieveImageLabels, "Could not retrieve ImageLabels (Docker image labels) from sous.Registry."),
		wrapError(db.restoreFromMetadata, "Could not determine cluster name based on SingularityDeploy Metadata."),
		wrapError(db.unpackDeployConfig, "Could not convert data from a SingularityDeploy to a sous.Deployment."),
		wrapError(db.determineManifestKind, "Could not determine SingularityRequestType."),
	)
}

func reqID(rp *dtos.SingularityRequestParent) (id string) {
	// defer func() { recover() }() because we explicitly do not care if this
	// panics. It is only used in certain low-level logs, and we don't mind
	// if we get some garbage data there. There is a fear that some race
	// condition between asserting that rp and rp.Request are not nil and
	// accessing their members may cause panics here. Please do not remove
	// this line before asserting somehow that this race condition does not
	// exist.
	defer func() { recover() }()
	id = "singularity.reqID() panicked"
	if rp == nil {
		return "<null RequestParent>"
	}
	if rp.Request == nil {
		return "<null Request>"
	}
	return rp.Request.Id
}

func (db *deploymentBuilder) basics() error {
	db.Target.Cluster = &sous.Cluster{BaseURL: db.req.SourceURL}
	db.Target.ExecutorData = &singularityTaskData{requestID: reqID(db.req.ReqParent)}
	Log.Vomit.Printf("Recording %v as requestID for instance.", db.Target.ExecutorData)
	db.request = db.req.ReqParent.Request
	db.reqID = reqID(db.req.ReqParent)
	return nil
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
func (db *deploymentBuilder) determineDeployStatus() error {
	logFDs("before determineDeployStatus()")
	defer logFDs("after determineDeployStatus()")

	rp := db.req.ReqParent
	if rp == nil {
		return malformedResponse{fmt.Sprintf("Singularity response didn't include a request parent. %v", db.req)}
	}

	rds := rp.RequestDeployState

	if rds == nil {
		return malformedResponse{"Singularity response didn't include a deploy state. ReqId: " + reqID(rp)}
	}

	if rds.PendingDeploy != nil {
		db.Target.Status = sous.DeployStatusPending
		db.depMarker = rds.PendingDeploy
	}
	// if there's no Pending deploy, we'll use the top of history in preference to Active
	// Consider: we might collect both and compare timestamps, but the active is
	// going to be the top of the history anyway unless there's been a more
	// recent failed deploy
	return nil
}

func (db *deploymentBuilder) retrieveDeployHistory() error {
	if db.depMarker == nil {
		return db.retrieveHistoricDeploy()
	}
	Log.Vomit.Printf("%q Getting deploy based on Pending marker.", db.reqID)
	return db.retrieveLiveDeploy()
}

func (db *deploymentBuilder) retrieveHistoricDeploy() error {
	Log.Vomit.Printf("%q Getting deploy from history", db.reqID)
	// !!! makes HTTP req
	if db.request == nil {
		return malformedResponse{"Singularity request parent had no request."}
	}
	sing := db.req.Sing
	depHistList, err := sing.GetDeploys(db.request.Id, 1, 1)
	Log.Vomit.Printf("%q Got history from Singularity with %d items.", db.reqID, len(depHistList))
	if err != nil {
		return errors.Wrap(err, "GetDeploys")
	}

	if len(depHistList) == 0 {
		return malformedResponse{"Singularity deploy history list was empty."}
	}

	partialHistory := depHistList[0]

	Log.Vomit.Printf("%q %#v", db.reqID, partialHistory)
	if partialHistory.DeployMarker == nil {
		return malformedResponse{"Singularity deploy history had no deploy marker."}
	}

	Log.Vomit.Printf("%q %#v", db.reqID, partialHistory.DeployMarker)
	db.depMarker = partialHistory.DeployMarker
	return db.retrieveLiveDeploy()
}

func (db *deploymentBuilder) retrieveLiveDeploy() error {
	// !!! makes HTTP req
	sing := db.req.Sing
	dh, err := sing.GetDeploy(db.depMarker.RequestId, db.depMarker.DeployId)
	if err != nil {
		Log.Vomit.Printf("%q received error retrieving history entry for deploy marker: %#v %#v", db.reqID, db.depMarker, err)
		return errors.Wrapf(err, "%q %#v", db.reqID, db.depMarker)
	}
	Log.Vomit.Printf("%q Deploy history entry retrieved: %#v", db.reqID, dh)

	db.history = dh

	return nil
}

func (db *deploymentBuilder) extractDeployFromDeployHistory() error {
	Log.Debug.Printf("%q Extracting deploy from history: %#v", db.reqID, db.history)
	db.deploy = db.history.Deploy
	if db.deploy == nil {
		return malformedResponse{"Singularity deploy history included no deploy"}
	}

	return nil
}

func (db *deploymentBuilder) sousDeployCheck() error {
	if cnl, ok := db.deploy.Metadata[sous.ClusterNameLabel]; ok {
		for _, cn := range db.clusters.Names() {
			if cnl == cn {
				return nil
			}
		}
		return notThisClusterError{cnl, db.clusters.Names()}
	}
	return nonSousError{}
}

func (db *deploymentBuilder) determineStatus() error {
	if db.history.DeployResult == nil {
		db.Target.Status = sous.DeployStatusPending
		return nil
	}
	if db.history.DeployResult.DeployState == dtos.SingularityDeployResultDeployStateSUCCEEDED {
		db.Target.Status = sous.DeployStatusActive
	} else {
		msg := db.history.DeployResult.Message
		if len(db.history.DeployResult.DeployFailures) > 0 {
			msgs := []string{}
			for _, df := range db.history.DeployResult.DeployFailures {
				msgs = append(msgs, df.Message)
			}
			msg = strings.Join(msgs, ", ")
		}

		db.Target.ExecutorMessage = fmt.Sprintf("Deploy faulure: %q %s/request/%s/deploy/%s",
			msg,
			db.req.SourceURL,
			db.history.Deploy.RequestId,
			db.history.Deploy.Id,
		)
		db.Target.Status = sous.DeployStatusFailed
	}

	return nil
}

func (db *deploymentBuilder) extractArtifactName() error {
	logFDs("before extractArtifactName()")
	defer logFDs("after extractArtifactName()")
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
	Log.Vomit.Printf("%q Labels: %v", db.reqID, labels)

	db.Target.SourceID, err = docker.SourceIDFromLabels(labels)
	if err != nil {
		return errors.Wrapf(malformedResponse{err.Error()}, "For reqID: %s", reqID(db.req.ReqParent))
	}

	return nil
}

func getMetadataField(field string, md map[string]string) (val string, err error) {
	var ok bool
	val, ok = md[field]
	if !ok {
		err = malformedResponse{fmt.Sprintf("Deploy Metadata did not include a %s", field)}
	}
	return
}

func (db *deploymentBuilder) restoreFromMetadata() error {
	var err error
	db.Target.ClusterName, err = getMetadataField(sous.ClusterNameLabel, db.deploy.Metadata)
	if err != nil {
		return err
	}

	// An absent flavor from the metadata is unseemly, but probably means that
	// the deploy predates flavor metadata handling
	// perhaps it's worth logging about this, or erroring on this and clobbering
	// old requests.
	//  - if you're debugging a deploy issue related to flavor, let's enforce
	//  this more strictly, and we'll deal with the fallout then -jdl
	db.Target.Flavor, _ = getMetadataField(sous.FlavorLabel, db.deploy.Metadata)
	return nil
}

func (db *deploymentBuilder) unpackDeployConfig() error {
	db.Target.Env = db.deploy.Env
	Log.Vomit.Printf("%q Env: %+v", db.reqID, db.deploy.Env)
	if db.Target.Env == nil {
		db.Target.Env = make(map[string]string)
	}

	singRez := db.deploy.Resources
	if singRez == nil {
		return malformedResponse{"Deploy object lacks resources field"}
	}
	db.Target.Resources = make(sous.Resources)
	db.Target.Resources["cpus"] = fmt.Sprintf("%f", singRez.Cpus)
	db.Target.Resources["memory"] = fmt.Sprintf("%f", singRez.MemoryMb)
	db.Target.Resources["ports"] = fmt.Sprintf("%d", singRez.NumPorts)

	db.Target.NumInstances = int(db.request.Instances)
	db.Target.Owners = make(sous.OwnerSet)
	for _, o := range db.request.Owners {
		db.Target.Owners.Add(o)
	}

	for _, v := range db.deploy.ContainerInfo.Volumes {
		db.Target.DeployConfig.Volumes = append(db.Target.DeployConfig.Volumes,
			&sous.Volume{
				Host:      v.HostPath,
				Container: v.ContainerPath,
				Mode:      sous.VolumeMode(v.Mode),
			})
	}
	Log.Vomit.Printf("%q Volumes %+v", db.reqID, db.Target.DeployConfig.Volumes)
	if len(db.Target.DeployConfig.Volumes) > 0 {
		Log.Debug.Printf("%q %+v", db.reqID, db.Target.DeployConfig.Volumes[0])
	}

	if db.deploy.Healthcheck != nil {
		db.Target.Startup.ConnectDelay = int(db.deploy.Healthcheck.StartupDelaySeconds)
		db.Target.Startup.Timeout = int(db.deploy.Healthcheck.StartupTimeoutSeconds)
		db.Target.Startup.ConnectInterval = int(db.deploy.Healthcheck.StartupIntervalSeconds)
		db.Target.Startup.CheckReadyProtocol = string(db.deploy.Healthcheck.Protocol)
		db.Target.Startup.CheckReadyURIPath = string(db.deploy.Healthcheck.Uri)
		db.Target.Startup.CheckReadyPortIndex = int(db.deploy.Healthcheck.PortIndex)
		db.Target.Startup.CheckReadyURITimeout = int(db.deploy.Healthcheck.ResponseTimeoutSeconds)
		db.Target.Startup.CheckReadyInterval = int(db.deploy.Healthcheck.IntervalSeconds)
		db.Target.Startup.CheckReadyRetries = int(db.deploy.Healthcheck.MaxRetries)

		db.Target.Startup.CheckReadyFailureStatuses = make([]int, len(db.deploy.Healthcheck.FailureStatusCodes))
		for n, code := range db.deploy.Healthcheck.FailureStatusCodes {
			db.Target.Startup.CheckReadyFailureStatuses[n] = int(code)
		}
	}

	return nil
}

func (db *deploymentBuilder) determineManifestKind() error {
	switch db.request.RequestType {
	default:
		return fmt.Errorf("Unrecognized request type returned by Singularity: %v", db.request.RequestType)
	case dtos.SingularityRequestRequestTypeSERVICE:
		db.Target.Kind = sous.ManifestKindService
	case dtos.SingularityRequestRequestTypeWORKER:
		db.Target.Kind = sous.ManifestKindWorker
	case dtos.SingularityRequestRequestTypeON_DEMAND:
		db.Target.Kind = sous.ManifestKindOnDemand
	case dtos.SingularityRequestRequestTypeSCHEDULED:
		db.Target.Kind = sous.ManifestKindScheduled
	case dtos.SingularityRequestRequestTypeRUN_ONCE:
		db.Target.Kind = sous.ManifestKindOnce
	}
	return nil
}
