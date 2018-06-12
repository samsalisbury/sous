package singularity

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/swaggering"
	"github.com/satori/go.uuid"
)

var illegalDeployIDChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)
var illegalRequestIDChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// SanitizeDeployID replaces characters forbidden in a Singularity deploy ID
// with underscores.
func SanitizeDeployID(in string) string {
	return illegalDeployIDChars.ReplaceAllString(in, "_")
}

// StripDeployID removes all characters forbidden in a Singularity deployID.
func StripDeployID(in string) string {
	return illegalDeployIDChars.ReplaceAllString(in, "")
}

func stripMetadata(in string) string {
	return strings.Split(in, "+")[0]
}

type (
	// RectiAgent is an implementation of the RectificationClient interface
	RectiAgent struct {
		singClients map[string]swaggering.Requester
		sync.RWMutex
		labeller sous.ImageLabeller
		log      logging.LogSink
	}

	singularityTaskData struct {
		requestID string
	}
)

// NewRectiAgent returns a set-up RectiAgent
func NewRectiAgent(l sous.ImageLabeller, ls logging.LogSink) *RectiAgent {
	return &RectiAgent{
		singClients: make(map[string]swaggering.Requester),
		labeller:    l,
		log:         ls,
	}
}

// mapResources produces a dtoMap appropriate for building a Singularity
// dto.Resources struct from
func mapResources(r sous.Resources) dtoMap {
	return dtoMap{
		"Cpus":     r.Cpus(),
		"MemoryMb": r.Memory(),
		"NumPorts": int32(r.Ports()),
	}
}

// Deploy sends requests to Singularity to make a deployment happen
func (ra *RectiAgent) Deploy(d sous.Deployable, reqID, depID string) error {
	if d.BuildArtifact == nil {
		return &sous.MissingImageNameError{Cause: fmt.Errorf("Missing BuildArtifact on Deployable")}
	}
	dockerImage := d.BuildArtifact.DigestReference
	clusterURI := d.Deployment.Cluster.BaseURL
	labels, err := ra.labeller.ImageLabels(dockerImage)
	if err != nil {
		return err
	}
	messages.ReportLogFieldsMessage("Build deploying instance", logging.DebugLevel, ra.log, d, reqID)

	depReq, err := buildDeployRequest(d, reqID, depID, labels, ra.log)
	if err != nil {
		return err
	}

	messages.ReportLogFieldsMessage("Sending Deploy req to singularity Client", logging.DebugLevel, ra.log, depReq)

	pathParamMap := map[string]interface{}{}

	user := "sous"
	if d.Deployment != nil && len(d.Deployment.User.Email) > 1 {
		user = "sous_" + d.Deployment.User.Email
	}
	queryParamMap := map[string]interface{}{
		"user": user,
	}

	//Note: Due to lack of way to pass User/queryParamMap easily, just lifting the singularityClient.Deploy code here
	response := new(dtos.SingularityRequestParent)

	err = ra.getSingularityRequester(clusterURI).DTORequest("singularity-deploy", response, "POST", "/api/deploys", pathParamMap, queryParamMap, depReq)

	if err != nil {
		messages.ReportLogFieldsMessage("Singularity client returned following error", logging.WarningLevel, ra.log, depReq, reqID, err, response)
	}
	return err
}

func buildDeployRequest(d sous.Deployable, reqID, depID string, metadata map[string]string, log logging.LogSink) (*dtos.SingularityDeployRequest, error) {
	var depReq swaggering.Fielder
	dockerImage := d.BuildArtifact.DigestReference
	r := d.Deployment.DeployConfig.Resources
	e := d.Deployment.DeployConfig.Env
	vols := d.Deployment.DeployConfig.Volumes

	metadata[sous.ClusterNameLabel] = d.Deployment.ClusterName
	metadata[sous.FlavorLabel] = d.Deployment.Flavor

	dockerInfo, err := swaggering.LoadMap(&dtos.SingularityDockerInfo{}, dtoMap{
		"Image":   dockerImage,
		"Network": dtos.SingularityDockerInfoSingularityDockerNetworkTypeBRIDGE, //defaulting to all bridge
	})
	if err != nil {
		return nil, err
	}

	res, err := swaggering.LoadMap(&dtos.Resources{}, mapResources(r))
	if err != nil {
		return nil, err
	}

	vs := dtos.SingularityVolumeList{}
	for _, v := range vols {
		if v == nil {
			messages.ReportLogFieldsMessage("nil volume", logging.WarningLevel, log)
			continue
		}
		sv, err := swaggering.LoadMap(&dtos.SingularityVolume{}, dtoMap{
			"ContainerPath": v.Container,
			"HostPath":      v.Host,
			"Mode":          dtos.SingularityVolumeSingularityDockerVolumeMode(string(v.Mode)),
		})
		if err != nil {
			return nil, err
		}
		vs = append(vs, sv.(*dtos.SingularityVolume))
	}

	ci, err := swaggering.LoadMap(&dtos.SingularityContainerInfo{}, dtoMap{
		"Type":    dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
		"Docker":  dockerInfo,
		"Volumes": vs,
	})
	if err != nil {
		return nil, err
	}

	user := "unknown_sous_deploy"
	if d.Deployment != nil && len(d.Deployment.User.Email) > 1 {
		user = d.Deployment.User.Email
	}
	message := fmt.Sprintf("Deployed by %s", user)
	depMap := dtoMap{
		"Id":            depID,
		"RequestId":     reqID,
		"Resources":     res,
		"ContainerInfo": ci,
		"Env":           map[string]string(e),
		"Metadata":      metadata,
	}

	if err := MapStartupIntoHealthcheckOptions((*map[string]interface{})(&depMap), d.Deployment.DeployConfig.Startup); err != nil {
		return nil, err
	}

	dep, err := swaggering.LoadMap(&dtos.SingularityDeploy{}, depMap)
	if err != nil {
		return nil, err
	}
	messages.ReportLogFieldsMessage("Deploy", logging.DebugLevel, log, dep, ci, dockerInfo)

	depReq, err = swaggering.LoadMap(&dtos.SingularityDeployRequest{}, dtoMap{"Deploy": dep, "Message": message})

	if err != nil {
		return nil, err
	}
	return depReq.(*dtos.SingularityDeployRequest), nil
}

// MapStartupIntoHealthcheckOptions updates the given dtoMap with fields for a
// HealthcheckOptions struct if appropriate.
// map[string]interface{} is used so that the function can be exported
// and used in integration tests. Once type aliases land, these backflips can go away.
func MapStartupIntoHealthcheckOptions(depMap *map[string]interface{}, startup sous.Startup) error {
	if startup.SkipCheck {
		return nil
	}

	hcMap := dtoMap{}

	hcMap["StartupDelaySeconds"] = int32(startup.ConnectDelay)
	hcMap["StartupTimeoutSeconds"] = int32(startup.Timeout)
	hcMap["StartupIntervalSeconds"] = int32(startup.ConnectInterval)
	failStatuses := make([]int32, len(startup.CheckReadyFailureStatuses))
	for n, c := range startup.CheckReadyFailureStatuses {
		failStatuses[n] = int32(c)
	}
	hcMap["FailureStatusCodes"] = failStatuses

	hcMap["Protocol"] = dtos.HealthcheckOptionsHealthcheckProtocol(startup.CheckReadyProtocol)
	hcMap["Uri"] = startup.CheckReadyURIPath
	hcMap["PortIndex"] = int32(startup.CheckReadyPortIndex)
	hcMap["ResponseTimeoutSeconds"] = int32(startup.CheckReadyURITimeout)
	hcMap["IntervalSeconds"] = int32(startup.CheckReadyInterval)
	hcMap["MaxRetries"] = int32(startup.CheckReadyRetries)

	hc, err := swaggering.LoadMap(&dtos.HealthcheckOptions{}, hcMap)
	(*depMap)["Healthcheck"] = hc
	return err
}

func singRequestFromDeployment(dep *sous.Deployment, reqID string, log logging.LogSink) (string, *dtos.SingularityRequest, error) {
	cluster := dep.Cluster.BaseURL
	instanceCount := dep.DeployConfig.NumInstances
	kind := dep.Kind
	owners := dep.Owners
	messages.ReportLogFieldsMessage("Creating application", logging.DebugLevel, log, cluster, reqID, instanceCount)
	reqType, err := determineRequestType(kind)
	if err != nil {
		return "", nil, err
	}
	reqFields := dtoMap{
		"Id":          reqID,
		"RequestType": reqType,
		"Instances":   int32(instanceCount),
		"Owners":      swaggering.StringList(owners.Slice()),
	}
	if reqType == dtos.SingularityRequestRequestTypeSCHEDULED {
		reqFields["Schedule"] = dep.Schedule

		// until and unless someone asks
		reqFields["ScheduleType"] = dtos.SingularityRequestScheduleTypeCRON

		// also present but not addressed:
		// taskExecutionTimeLimitMillis
	}
	req, err := swaggering.LoadMap(&dtos.SingularityRequest{}, reqFields)

	if err != nil {
		return "", nil, err
	}

	return cluster, req.(*dtos.SingularityRequest), nil
}

// PostRequest sends requests to Singularity to create a new Request
func (ra *RectiAgent) PostRequest(d sous.Deployable, reqID string) error {
	cluster, req, err := singRequestFromDeployment(d.Deployment, reqID, ra.log)
	if err != nil {
		return err
	}

	messages.ReportLogFieldsMessage("Create Request", logging.DebugLevel, ra.log, req)
	_, err = ra.singularityClient(cluster).PostRequest(req)
	return err
}

func determineRequestType(kind sous.ManifestKind) (dtos.SingularityRequestRequestType, error) {
	switch kind {
	default:
		return dtos.SingularityRequestRequestType(""), fmt.Errorf("Unrecognized Sous manifest kind: %v", kind)
	case sous.ManifestKindService:
		return dtos.SingularityRequestRequestTypeSERVICE, nil
	case sous.ManifestKindWorker:
		return dtos.SingularityRequestRequestTypeWORKER, nil
	case sous.ManifestKindOnDemand:
		return dtos.SingularityRequestRequestTypeON_DEMAND, nil
	case sous.ManifestKindScheduled:
		return dtos.SingularityRequestRequestTypeSCHEDULED, nil
	case sous.ManifestKindOnce:
		return dtos.SingularityRequestRequestTypeRUN_ONCE, nil
	}
}

// DeleteRequest sends a request to Singularity to delete a request
func (ra *RectiAgent) DeleteRequest(cluster, reqID, message string) error {
	messages.ReportLogFieldsMessage("Deleting application", logging.DebugLevel, ra.log, cluster, reqID, message)
	req, err := swaggering.LoadMap(&dtos.SingularityDeleteRequestRequest{}, dtoMap{
		"Message": "Sous: " + message,
	})
	if err != nil {
		return err
	}

	messages.ReportLogFieldsMessage("Delete req", logging.DebugLevel, ra.log, req)
	_, err = ra.singularityClient(cluster).DeleteRequest(reqID,
		req.(*dtos.SingularityDeleteRequestRequest))
	return err
}

// Scale sends requests to Singularity to change the number of instances
// running for a given Request
func (ra *RectiAgent) Scale(cluster, reqID string, instanceCount int, message string) error {
	messages.ReportLogFieldsMessage("Scaling", logging.DebugLevel, ra.log, cluster, reqID, instanceCount, message)
	sr, err := swaggering.LoadMap(&dtos.SingularityScaleRequest{}, dtoMap{
		"ActionId": "SOUS_RECTIFY_" + StripDeployID(uuid.NewV4().String()), // not positive this is appropriate
		// omitting DurationMillis - bears discussion
		"Instances":        int32(instanceCount),
		"Message":          "Sous" + message,
		"SkipHealthchecks": false,
	})

	if err != nil {
		return err
	}

	messages.ReportLogFieldsMessage("Scale req", logging.DebugLevel, ra.log, sr)
	_, err = ra.singularityClient(cluster).Scale(reqID, sr.(*dtos.SingularityScaleRequest))
	return err
}

//Deploy is actually not going to use traditional client, it will use Requester interface
//Which on normal runs comes from Singularity, but testing gets injected (via the map) for testing
//It also doesn't call the normal wrapper Client, hence the call straight to DTORequest
func (ra *RectiAgent) getSingularityRequester(url string) swaggering.Requester {
	ra.RLock()
	defer ra.RUnlock()
	if _, ok := ra.singClients[url]; !ok {
		c := singularity.NewClient(url)
		ra.singClients[url] = c
	}

	return ra.singClients[url]
}

func (ra *RectiAgent) getSingularityClient(url string) (*singularity.Client, bool) {
	ra.RLock()
	defer ra.RUnlock()
	r, ok := ra.singClients[url]

	var o *singularity.Client
	if ok {
		o = r.(*singularity.Client)
	}

	return o, ok
}

func (ra *RectiAgent) singularityClient(url string) *singularity.Client {
	cl, ok := ra.getSingularityClient(url)
	if ok {
		return cl
	}
	ra.Lock()
	defer ra.Unlock()
	cl = singularity.NewClient(url)
	//cl.Debug = true
	ra.singClients[url] = cl
	return cl
}
