package singularity

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/firsterr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
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
		log       logging.LogSink
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

func ignorableDeploy(log logging.LogSink, err error) bool {
	switch errors.Cause(err).(type) {
	case nonSousError, notThisClusterError:
		return true
	}
	return false
}

func (mr malformedResponse) Error() string {
	return mr.message
}

func isMalformed(log logging.LogSink, err error) bool {
	err = errors.Cause(err)
	_, isMal := err.(malformedResponse)
	_, isUMT := err.(*json.UnmarshalTypeError)
	_, isUST := err.(*json.UnsupportedTypeError)
	_, isUSV := err.(*json.UnsupportedValueError)
	return isMal || isUMT || isUST || isUSV
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
	return !isMalformed(db.log, err) &&
		!ignorableDeploy(db.log, err) &&
		db.req.SourceURL != "" &&

		db.req.ReqParent != nil &&
		db.req.ReqParent.Request != nil &&
		db.req.ReqParent.Request.Id != ""
}

// BuildDeployment does all the work to collect the data for a Deployment
// from Singularity based on the initial SingularityRequest.
func BuildDeployment(reg sous.ImageLabeller, clusters sous.Clusters, req SingReq, log logging.LogSink) (sous.DeployState, error) {
	messages.ReportLogFieldsMessage("Build Deployment", logging.ExtraDebug1Level, log, req.ReqParent)
	db := deploymentBuilder{registry: reg, clusters: clusters, req: req, log: log}
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
		wrapError(db.getFullRequestParent, "Failed to retrieve full RequestParent DTO."),
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
		wrapError(db.extractSchedule, "Could not determine Singularity schedule."),
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
	messages.ReportLogFieldsMessage("Recording as requestID for instance.", logging.ExtraDebug1Level, db.log, db.Target.ExecutorData)
	db.request = db.req.ReqParent.Request
	db.reqID = reqID(db.req.ReqParent)
	return nil
}

func (db *deploymentBuilder) getFullRequestParent() error {
	if db.req.ReqParent.ActiveDeploy != nil || db.req.ReqParent.PendingDeploy != nil {
		// already have useful info
		return nil
	}

	rp, err := db.req.Sing.GetRequest(db.reqID, false)
	if err != nil {
		return err
	}
	db.req.ReqParent = rp
	return nil
}

// If there is a Pending deploy, as far as Sous is concerned, that's "to
// come" - we optimistically assume it will become Active, and that's the
// Deployment we should consider live.
func (db *deploymentBuilder) determineDeployStatus() error {
	rp := db.req.ReqParent
	if rp == nil {
		return malformedResponse{fmt.Sprintf("Singularity response didn't include a request parent. %v", db.req)}
	}

	rds := rp.RequestDeployState

	if rds == nil {
		return malformedResponse{"Singularity response didn't include a deploy state. ReqId: " + reqID(rp)}
	}

	switch {
	default:
		db.Target.Status = sous.DeployStatusAny
	case rp.State != dtos.SingularityRequestParentRequestStateACTIVE &&
		rp.State != dtos.SingularityRequestParentRequestStateDEPLOYING_TO_UNPAUSE:
		db.Target.Status = sous.DeployStatusAny
	case rds.PendingDeploy != nil:
		db.Target.Status = sous.DeployStatusPending
		db.depMarker = rds.PendingDeploy
		db.deploy = rp.PendingDeploy
		/*
			XXX(jdl) This doesn't work, because as of 0.19, S9y Request responses
			don't include enough information to distinguish successfully deployed
			requests from fallbacks. There are promising fields in 0.20, so we
			should revisit.

			case rds.ActiveDeploy != nil:
				db.Target.Status = sous.DeployStatusActive
				db.depMarker = rds.ActiveDeploy
				db.deploy = rp.ActiveDeploy
		*/
	}
	return nil
}

func (db *deploymentBuilder) retrieveDeployHistory() error {
	if db.depMarker == nil {
		return db.retrieveHistoricDeploy()
	}
	messages.ReportLogFieldsMessage("Getting deploy based on Pending marker.", logging.ExtraDebug1Level, db.log, db.reqID)
	return db.retrieveLiveDeploy()
}

func (db *deploymentBuilder) retrieveHistoricDeploy() error {
	messages.ReportLogFieldsMessage("Getting deploy from history", logging.ExtraDebug1Level, db.log, db.reqID)
	// !!! makes HTTP req
	if db.request == nil {
		return malformedResponse{"Singularity request parent had no request."}
	}
	sing := db.req.Sing
	depHistList, err := sing.GetDeploys(db.request.Id, 1, 1)
	messages.ReportLogFieldsMessage("Got history from Singularity with items.", logging.ExtraDebug1Level, db.log, db.reqID, len(depHistList))
	if err != nil {
		return errors.Wrap(err, "GetDeploys")
	}

	if len(depHistList) == 0 {
		return malformedResponse{"Singularity deploy history list was empty."}
	}

	partialHistory := depHistList[0]

	messages.ReportLogFieldsMessage("Partial history.", logging.ExtraDebug1Level, db.log, db.reqID, partialHistory)
	if partialHistory.DeployMarker == nil {
		return malformedResponse{"Singularity deploy history had no deploy marker."}
	}

	messages.ReportLogFieldsMessage("Partial history DeployMarker.", logging.ExtraDebug1Level, db.log, db.reqID, partialHistory.DeployMarker)
	db.depMarker = partialHistory.DeployMarker
	return db.retrieveLiveDeploy()
}

func (db *deploymentBuilder) retrieveLiveDeploy() error {
	if db.deploy != nil {
		// handled already
		return nil
	}
	// !!! makes HTTP req
	sing := db.req.Sing
	dh, err := sing.GetDeploy(db.depMarker.RequestId, db.depMarker.DeployId)
	if err != nil {
		messages.ReportLogFieldsMessage("Received error retrieving history entry for deploy marker.", logging.ExtraDebug1Level, db.log, db.reqID, db.depMarker, err)
		return errors.Wrapf(err, "%q %#v", db.reqID, db.depMarker)
	}

	messages.ReportLogFieldsMessage("Deploy history entry retrieved.", logging.ExtraDebug1Level, db.log, db.reqID, dh)
	db.history = dh

	return nil
}

func (db *deploymentBuilder) extractDeployFromDeployHistory() error {
	if db.deploy != nil {
		// have a deploy already from the request
		return nil
	}
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
				messages.ReportLogFieldsMessage("Deploy cluster found in clusters.", logging.ExtraDebug1Level, db.log, cnl, db.clusters)
				return nil
			}
		}
		return notThisClusterError{cnl, db.clusters.Names()}
	}
	return nonSousError{}
}

func (db *deploymentBuilder) determineStatus() error {
	if db.Target.Status != sous.DeployStatusAny {
		// handled already
		return nil
	}
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

		db.Target.ExecutorMessage = fmt.Sprintf("Deploy failure: %q %s/request/%s/deploy/%s",
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
	// XXX for unlabelled images, we need to handle this somehow...
	labels, err := db.registry.ImageLabels(db.imageName)
	if err != nil {
		return malformedResponse{err.Error()}
	}

	messages.ReportLogFieldsMessage("Labels", logging.ExtraDebug1Level, db.log, db.reqID, labels)
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
	messages.ReportLogFieldsMessage("UnpackDeployConfig", logging.ExtraDebug1Level, db.log, db.reqID, db.deploy.Env)
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
	messages.ReportLogFieldsMessage("Volumes", logging.ExtraDebug1Level, db.log, db.reqID, db.Target.DeployConfig.Volumes)
	if len(db.Target.DeployConfig.Volumes) > 0 {
		messages.ReportLogFieldsMessage("UnpackDeployConfig volume 0", logging.DebugLevel, db.log, db.reqID, db.Target.DeployConfig.Volumes[0])
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
	} else {
		db.Target.Startup.SkipCheck = true
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

func (db *deploymentBuilder) extractSchedule() error {
	if db.Target.Kind == sous.ManifestKindScheduled {
		if db.request == nil {
			return fmt.Errorf("request is nil")
		}
		db.Target.DeployConfig.Schedule = db.request.Schedule
	}
	return nil
}
