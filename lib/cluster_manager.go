package sous

import "github.com/nyarly/spies"

type (
	// ClusterManager reads and writes deployments as scoped by cluster
	ClusterManager interface {
		ReadCluster(clusterName string) (Deployments, error)
		WriteCluster(clusterName string, deps Deployments) error
	}

	clusterManagerSpy struct {
		*spies.Spy
	}
)

// NewClusterManagerSpy produces a spy/controller pair for ClusterManager
func NewClusterManagerSpy() (ClusterManager, *spies.Spy) {
	spy := &spies.Spy{}

	return clusterManagerSpy{spy}, spy
}

func (cm clusterManagerSpy) ReadCluster(clusterName string) (Deployments, error) {
	res := cm.Called(clusterName)
	return res.Get(0).(Deployments), res.Error(1)
}

func (cm clusterManagerSpy) WriteCluster(clusterName string, deps Deployments) error {
	return cm.Called(clusterName, deps).Error(0)
}
