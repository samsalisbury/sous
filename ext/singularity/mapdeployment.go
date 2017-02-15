package singularity

import (
	"fmt"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
)

func mapDeployHistoryToDeployment(sid sous.SourceID,
	sr *dtos.SingularityRequest,
	dh *dtos.SingularityDeployHistory) (*sous.Deployment, error) {

	if dh.Deploy == nil {
		return nil, fmt.Errorf("deploy history contains no deployment")
	}

	// DeployConfig
	deployConfig, err := mapDeployHistoryToDeployConfig(dh.Deploy)
	if err != nil {
		return nil, err
	}

	// Owners
	owners := sous.OwnerSet{}
	for _, o := range sr.Owners {
		owners.Add(o)
	}

	// NumInstances
	numInstances := sr.Instances

	return &sous.Deployment{
		DeployConfig: *deployConfig,
		Owners:       owners,
		SourceID:     sid,
	}, nil
}

func mapDeployHistoryToDeployConfig(deploy *dtos.SingularityDeploy) (*sous.DeployConfig, error) {

	dc := sous.DeployConfig{}

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
		"cpus":   fmt.Sprint(deploy.Resources.Cpus),
		"memory": fmt.Sprint(deploy.Resources.MemoryMb),
		"ports":  fmt.Sprint(deploy.Resources.NumPorts),
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
		Env:       env,
		Resources: resources,
		Volumes:   volumes,
	}, nil
}
