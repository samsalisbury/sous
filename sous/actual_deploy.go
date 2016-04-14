package sous

import (
	"fmt"
	"log"
	"sync"

	"github.com/opentable/singularity"
	"github.com/opentable/singularity/dtos"
)

type ActualDeploy struct {
}

const ReqsPerServer = 10

func GetRunningDeploymentSet(singularities []*singularity.Client, registryUrl string) (deps Deployments) {
	errCh := make(chan error, 1)
	reqCh := make(chan *dtos.SingularityRequestParent, len(singularities)*ReqsPerServer)
	depCh := make(chan Deployment, ReqsPerServer)

	var singWait, depWait sync.WaitGroup

	singWait.Add(len(singularities))
	for _, singularity := range singularities {
		go getReqs(singularity, reqCh, errCh)
	}

	go handleSingErrs(errCh)

	go depPipeline(depWait, reqCh, depCh)

	for dep := range depCh {
		depWait.Done()
		deps = append(deps, dep)
	}
	return
}

func handleSingErrs(errs chan error) {
	for err := range errs {
		log.Print(err)
	}
}

func depPipeline(wait sync.WaitGroup, reqCh chan *dtos.SingularityRequestParent, registryClient string, depCh chan Deployment) {
	var once sync.Once
	for req := range reqCh {
		wait.Add(1)
		once.Do(func() {
			go func() {
				wait.Wait()
				close(reqCh)
				close(depCh)
			}()
		})
		go deploymentFromRequest(req, registryClient, depCh)
	}
}

func getReqs(client *singularity.Client, reqCh chan *dtos.SingularityRequestParent, errCh chan error) {
	requests, err := client.GetRequests()
	if err != nil {
		errCh <- err
		return
	}
	for _, req := range requests {
		reqCh <- req
	}
}

/*
type SingularityRequestParent struct {
	ActiveDeploy             *SingularityDeploy
	PendingDeploy            *SingularityDeploy
	PendingDeployState       *SingularityPendingDeploy
	Request                  *SingularityRequest
	RequestDeployState       *SingularityRequestDeployState
	//	State *RequestState
}

type SingularityDeploy struct {
	Id                        string

	Command                               string
	Arguments                             StringList
	AutoAdvanceDeploySteps                bool
	ConsiderHealthyAfterRunningForSeconds int64
	ContainerInfo                         *SingularityContainerInfo
	CustomExecutorCmd                     string
	CustomExecutorId                      string
	CustomExecutorResources               *Resources
	CustomExecutorSource                  string
	CustomExecutorUser                    string
	DeployHealthTimeoutSeconds            int64
	DeployInstanceCountPerStep            int32
	DeployStepWaitTimeMs                  int32
	Env                                   map[string]string
	ExecutorData                          *ExecutorData
	HealthcheckIntervalSeconds            int64
	HealthcheckMaxRetries                 int32
	HealthcheckMaxTotalTimeoutSeconds     int64
	HealthcheckPortIndex                  int32
	HealthcheckTimeoutSeconds int64
	HealthcheckUri            string
	Labels                    map[string]string
	LoadBalancerGroups        StringList
	LoadBalancerPortIndex int32
	MaxTaskRetries        int32
	Metadata              map[string]string
	RequestId             string
	ServiceBasePath          string
	SkipHealthchecksOnDeploy bool
	Timestamp                int64
	Uris                     StringList
	Version                  string
}

type SingularityContainerInfo struct {
	Docker *SingularityDockerInfo
	//	Type *SingularityContainerType
	Volumes SingularityVolumeList
}

type SingularityDockerInfo struct {
	ForcePullImage bool
	Image          string
	//	Network *SingularityDockerNetworkType
	Parameters   map[string]string
	PortMappings SingularityDockerPortMappingList
	Privileged   bool
}

type Resources struct {
	Cpus     float64
	MemoryMb float64
	NumPorts int32
}

type SingularityRequest struct {
	//	RequestType *RequestType
	//	ScheduleType *ScheduleType
	//	SlavePlacement *SlavePlacement
	AllowedSlaveAttributes map[string]string
	BounceAfterScale       bool
	Group                                 string
	Id                                    string
	Instances                             int32
	KillOldNonLongRunningTasksAfterMillis int64
	LoadBalanced                          bool
	NumRetriesOnFailure                   int32
	Owners                                StringList
	QuartzSchedule                        string
	RackAffinity                          StringList
	RackSensitive                         bool
	ReadOnlyGroups                        StringList
	RequiredSlaveAttributes map[string]string
	Schedule                string
	ScheduledExpectedRuntimeMillis int64
	SkipHealthchecks               bool
	WaitAtLeastMillisAfterTaskFinishesForReschedule int64
}
*/

/*
Deployment struct {
	DeploymentConfig struct {
		Resources Resources
		//   map[string]string
		Env map[string]string
		NumInstances int
	}

	Cluster string
	NamedVersion
	NamedVersion struct {
		RepositoryName
		semv.Version
		Path
	}

	Owners []string
	Kind ManifestKind
}
*/
func deploymentFromRequest(cluster string, req *dtos.SingularityRequestParent, registryUrl string, depCh chan Deployment) {
	rezzes := make(Resources)
	singDep := req.ActiveDeploy
	singReq := req.Request
	singRez := singDep.CustomExecutorResources
	rezzes["cpus"] = fmt.Sprintf("%d", singRez.Cpus)
	rezzes["memory"] = fmt.Sprintf("%d", singRez.MemoryMb)
	rezzes["ports"] = fmt.Sprintf("%d", singRez.NumPorts)
	env := singDep.Env
	numinst := int(singReq.Instances)
	owners := singReq.Owners
	kind := ManifestKind("service")
	// Singularity's RequestType is an enum of
	//   SERVICE, WORKER, SCHEDULED, ON_DEMAND, RUN_ONCE;
	// Their annotations don't successfully list RequestType into their swagger.
	// Need an approach to handling the type -
	//   either manual Go structs that the resolving context is informed of OR
	//   additional processing of the swagger JSON files before we process it

	imageName := singDep.ContainerInfo.Docker.Image

	labels, err := LabelsForTaggedImage(registryUrl, repositoryName, tag)

	// We should probably check what kind of Container it is...
	// Which has a similar problem to RequestType - another Java enum that isn't Swaggered

	depCh <- Deployment{
		Cluster: cluster,

		Resources:    rezzes,
		Env:          env,
		NumInstances: numinst,
		Owners:       owners,
		Kind:         kind,

		RepositoryName: repo,
		Version:        version,
		Path:           path,
	}
}
