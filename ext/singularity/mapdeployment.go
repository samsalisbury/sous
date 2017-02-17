package singularity

import (
	"fmt"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
)

func mapDeployHistoryToDeployment(
	cluster sous.Cluster,
	sid sous.SourceID,
	sr *dtos.SingularityRequest,
	dh *dtos.SingularityDeployHistory) (*sous.Deployment, error) {

	if dh.Deploy == nil {
		return nil, fmt.Errorf("deploy history contains no deployment")
	}

	// DeployConfig
	deployConfig, err := mapDeployHistoryToDeployConfig(sr, dh.Deploy)
	if err != nil {
		return nil, err
	}

	// Owners
	owners := sous.OwnerSet{}
	for _, o := range sr.Owners {
		owners.Add(o)
	}

	// Kind
	kind, err := mapManifestKind(sr.RequestType)
	if err != nil {
		return nil, err
	}

	return &sous.Deployment{
		Cluster: &cluster,
		// TODO: Remove ClusterName from sous.Deployment and use Cluster.Name.
		ClusterName:  cluster.Name,
		DeployConfig: *deployConfig,
		Owners:       owners,
		SourceID:     sid,
		Kind:         kind,
	}, nil
}

func mapDeployHistoryToDeployConfig(req *dtos.SingularityRequest, deploy *dtos.SingularityDeploy) (*sous.DeployConfig, error) {

	// Env
	env := deploy.Env
	if env == nil {
		env = map[string]string{}
	}

	// Resources
	if deploy.Resources == nil {
		return nil, fmt.Errorf("deploy object lacks resources field")
	}
	resources := sous.Resources{
		"cpus":   fmt.Sprintf("%f", deploy.Resources.Cpus),
		"memory": fmt.Sprintf("%f", deploy.Resources.MemoryMb),
		"ports":  fmt.Sprintf("%d", deploy.Resources.NumPorts),
	}

	// Volumes
	var volumes sous.Volumes
	if deploy.ContainerInfo != nil && deploy.ContainerInfo.Volumes != nil {
		for _, v := range deploy.ContainerInfo.Volumes {
			volumes = append(volumes,
				&sous.Volume{
					Host:      v.HostPath,
					Container: v.ContainerPath,
					Mode:      sous.VolumeMode(v.Mode),
				})
		}
	}

	return &sous.DeployConfig{
		NumInstances: int(req.Instances),
		Env:          env,
		Resources:    resources,
		Volumes:      volumes,
	}, nil
}

func mapManifestKind(rt dtos.SingularityRequestRequestType) (sous.ManifestKind, error) {
	switch rt {
	default:
		return sous.ManifestKind(""), fmt.Errorf("unrecognised request type: %s", rt)
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
