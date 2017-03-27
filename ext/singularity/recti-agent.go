package singularity

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/swaggering"
	"github.com/satori/go.uuid"
)

var illegalDeployIDChars = regexp.MustCompile(`[^a-z|^A-Z|^0-9|^_]`)

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

// RectiAgent is an implementation of the RectificationClient interface
type RectiAgent struct {
	singClients map[string]*singularity.Client
	sync.RWMutex
	labeller sous.ImageLabeller
}

// NewRectiAgent returns a set-up RectiAgent
func NewRectiAgent(l sous.ImageLabeller) *RectiAgent {
	return &RectiAgent{
		singClients: make(map[string]*singularity.Client),
		labeller:    l,
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
func (ra *RectiAgent) Deploy(d sous.Deployable, reqID string) error {
	if d.BuildArtifact == nil {
		return &sous.MissingImageNameError{Cause: fmt.Errorf("Missing BuildArtifact on Deployable")}
	}
	dockerImage := d.BuildArtifact.Name
	clusterURI := d.Deployment.Cluster.BaseURL
	labels, err := ra.labeller.ImageLabels(dockerImage)
	if err != nil {
		return err
	}

	Log.Debug.Printf("Deploying instance %#v to request %s", d, reqID)
	depReq, err := buildDeployRequest(d, reqID, labels)
	if err != nil {
		return err
	}

	Log.Debug.Printf("Deploy req: %+ v", depReq)
	_, err = ra.singularityClient(clusterURI).Deploy(depReq)
	return err
}

func buildDeployRequest(d sous.Deployable, reqID string, metadata map[string]string) (*dtos.SingularityDeployRequest, error) {
	var depReq swaggering.Fielder
	depID := computeDeployID(&d)
	dockerImage := d.BuildArtifact.Name
	r := d.Deployment.DeployConfig.Resources
	e := d.Deployment.DeployConfig.Env
	vols := d.Deployment.DeployConfig.Volumes
	clusterName := d.Deployment.ClusterName
	flavor := d.Deployment.Flavor

	metadata[sous.SingularityDeployMetadataClusterName] = clusterName
	metadata[sous.SingularityDeployMetadataFlavor] = flavor

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
			Log.Warn.Printf("nil volume")
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

	dep, err := swaggering.LoadMap(&dtos.SingularityDeploy{}, dtoMap{
		"Id":                         depID,
		"RequestId":                  reqID,
		"Resources":                  res,
		"ContainerInfo":              ci,
		"Env":                        map[string]string(e),
		"Metadata":                   metadata,
		"DeployHealthTimeoutSeconds": int64(sous.SingularityDeployTimeout),
	})
	if err != nil {
		return nil, err
	}
	Log.Debug.Printf("Deploy: %+ v", dep)
	Log.Debug.Printf("  Container: %+ v", ci)
	Log.Debug.Printf("  Docker: %+ v", dockerInfo)
	depReq, err = swaggering.LoadMap(&dtos.SingularityDeployRequest{}, dtoMap{"Deploy": dep})
	if err != nil {
		return nil, err
	}
	return depReq.(*dtos.SingularityDeployRequest), nil
}

// PostRequest sends requests to Singularity to create a new Request
func (ra *RectiAgent) PostRequest(d sous.Deployable, reqID string) error {
	cluster := d.Deployment.Cluster.BaseURL
	instanceCount := d.Deployment.DeployConfig.NumInstances
	kind := d.Deployment.Kind
	owners := d.Deployment.Owners
	Log.Debug.Printf("Creating application %s %s %d", cluster, reqID, instanceCount)
	reqType, err := determineRequestType(kind)
	if err != nil {
		return err
	}
	req, err := swaggering.LoadMap(&dtos.SingularityRequest{}, dtoMap{
		"Id":          reqID,
		"RequestType": reqType,
		"Instances":   int32(instanceCount),
		"Owners":      swaggering.StringList(owners.Slice()),
	})

	if err != nil {
		return err
	}

	Log.Debug.Printf("Create Request: %+ v", req)
	_, err = ra.singularityClient(cluster).PostRequest(req.(*dtos.SingularityRequest))
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
	Log.Debug.Printf("Deleting application %s %s %s", cluster, reqID, message)
	req, err := swaggering.LoadMap(&dtos.SingularityDeleteRequestRequest{}, dtoMap{
		"Message": "Sous: " + message,
	})
	if err != nil {
		return err
	}

	Log.Debug.Printf("Delete req: %+ v", req)
	_, err = ra.singularityClient(cluster).DeleteRequest(reqID,
		req.(*dtos.SingularityDeleteRequestRequest))
	return err
}

// Scale sends requests to Singularity to change the number of instances
// running for a given Request
func (ra *RectiAgent) Scale(cluster, reqID string, instanceCount int, message string) error {
	Log.Debug.Printf("Scaling %s %s %d %s", cluster, reqID, instanceCount, message)
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

	Log.Debug.Printf("Scale req: %+ v", sr)
	_, err = ra.singularityClient(cluster).Scale(reqID, sr.(*dtos.SingularityScaleRequest))
	return err
}

func (ra *RectiAgent) getSingularityClient(url string) (*singularity.Client, bool) {
	ra.RLock()
	defer ra.RUnlock()
	cl, ok := ra.singClients[url]
	return cl, ok
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
