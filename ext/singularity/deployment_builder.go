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
		clusters  sous.Clusters
		Target    sous.DeployState
		imageName string
		depMarker sDepMarker
		history   sHistory
		deploy    sDeploy
		request   sRequest
		req       SingReq
		registry  sous.ImageLabeller
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
	_, yes := err.(malformedResponse)
	Log.Vomit.Printf("err: %+v %T %t", err, err, yes)
	return yes
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

	db.Target.Cluster = &sous.Cluster{BaseURL: req.SourceURL}
	db.request = req.ReqParent.Request

	return db.Target, db.canRetry(db.completeConstruction())
}

func (db *deploymentBuilder) completeConstruction() error {
	wrapError := func(fn func() error, msgStr string) func() error {
		return func() error {
			return errors.Wrap(fn(), msgStr)
		}
	}
	return firsterr.Returned(
		wrapError(db.determineDeployStatus, "Failed to determine deploy status."),
		//FIXME maybe db.retrieveDeploy should be retrieveDeployHistory?
		wrapError(db.retrieveDeploy, "Failed to retrieve SingularityDeployHistory from SingularityRequestParent."),
		wrapError(db.extractDeployFromDeployHistory, "Failed to extract SingularityDeploy from SingularityDeployHistory."),
		wrapError(db.sousDeployCheck, "Could not determine if the SingularityDeploy is controlled by Sous"),
		wrapError(db.determineStatus, "Could not determine current status of SingularityDeploy"),
		wrapError(db.extractArtifactName, "Could not extract ArtifactName (Docker image name) from SingularityDeploy."),
		wrapError(db.retrieveImageLabels, "Could not retrieve ImageLabels (Docker image labels) from sous.Registry."),
		wrapError(db.assignClusterName, "Could not determine cluster name based on SingularityDeploy Metadata."),
		wrapError(db.unpackDeployConfig, "Could not convert data from a SingularityDeploy to a sous.Deployment."),
		wrapError(db.determineManifestKind, "Could not determine SingularityRequestType."),
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

func (db *deploymentBuilder) retrieveDeploy() error {
	if db.depMarker == nil {
		return db.retrieveHistoricDeploy()
	}
	Log.Vomit.Printf("Getting deploy based on Pending marker.")
	return db.retrieveLiveDeploy()
}

func (db *deploymentBuilder) retrieveHistoricDeploy() error {
	Log.Vomit.Printf("Getting deploy from history")
	// !!! makes HTTP req
	if db.request == nil {
		return malformedResponse{"Singularity request parent had no request."}
	}
	sing := db.req.Sing
	depHistList, err := sing.GetDeploys(db.request.Id, 1, 1)
	Log.Vomit.Printf("Got history from Singularity with %d items.", len(depHistList))
	if err != nil {
		return errors.Wrap(err, "GetDeploys")
	}

	if len(depHistList) == 0 {
		return malformedResponse{"Singularity deploy history list was empty."}
	}

	partialHistory := depHistList[0]

	Log.Vomit.Printf("%#v", partialHistory)
	if partialHistory.DeployMarker == nil {
		return malformedResponse{"Singularity deploy history had no deploy marker."}
	}

	Log.Vomit.Printf("%#v", partialHistory.DeployMarker)
	db.depMarker = partialHistory.DeployMarker
	db.retrieveLiveDeploy()
	return nil
}

func (db *deploymentBuilder) retrieveLiveDeploy() error {
	// !!! makes HTTP req
	sing := db.req.Sing
	dh, err := sing.GetDeploy(db.depMarker.RequestId, db.depMarker.DeployId)
	if err != nil {
		return errors.Wrapf(err, "%#v", db.depMarker)
	}
	Log.Vomit.Printf("Deploy history entry retrieved: %#v", dh)

	db.history = dh

	return nil
}

func (db *deploymentBuilder) extractDeployFromDeployHistory() error {
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
	Log.Vomit.Print("Labels: ", labels)

	db.Target.SourceID, err = docker.SourceIDFromLabels(labels)
	if err != nil {
		return errors.Wrapf(malformedResponse{err.Error()}, "For reqID: %s", reqID(db.req.ReqParent))
	}

	return nil
}

func (db *deploymentBuilder) assignClusterName() error {
	var ok bool
	db.Target.ClusterName, ok = db.deploy.Metadata[sous.ClusterNameLabel]
	if !ok {
		return malformedResponse{fmt.Sprintf("Deploy Metadata did not include a %s", sous.ClusterNameLabel)}
	}
	return nil
}

func (db *deploymentBuilder) unpackDeployConfig() error {
	db.Target.Env = db.deploy.Env
	Log.Vomit.Printf("Env: %+v", db.deploy.Env)
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
	Log.Vomit.Printf("Volumes %+v", db.Target.DeployConfig.Volumes)
	if len(db.Target.DeployConfig.Volumes) > 0 {
		Log.Debug.Printf("%+v", db.Target.DeployConfig.Volumes[0])
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
