package sous

import (
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

// A DispatchStateManager handles dispatching data requests to local or remote datastores.
type DispatchStateManager struct {
	local   StateManager
	remotes map[string]ClusterManager
	log     logging.LogSink
}

// NewDispatchStateManager builds a DispatchStateManager.
func NewDispatchStateManager(localCluster string, clusters []string, local StateManager, remote ClusterManager, ls logging.LogSink) *DispatchStateManager {
	dsm := &DispatchStateManager{
		local:   local,
		remotes: map[string]ClusterManager{},
		log:     ls,
	}
	for _, n := range clusters {
		dsm.remotes[n] = remote
	}
	switch lcm := local.(type) {
	default:
		dsm.remotes[localCluster] = MakeClusterManager(local, ls)
	case ClusterManager:
		dsm.remotes[localCluster] = lcm
	}
	return dsm
}

// ReadState implements StateManager on DispatchStateManager.
func (dsm *DispatchStateManager) ReadState() (*State, error) {
	baseState, err := dsm.local.ReadState()
	if err != nil {
		return nil, errors.Wrapf(err, "base state")
	}
	for cluster, manager := range dsm.remotes {
		c, err := manager.ReadCluster(cluster)
		if err != nil {
			return nil, errors.Wrapf(err, cluster)
		}
		ds := []*Deployment{}
		for _, d := range c.Snapshot() {
			ds = append(ds, d)
		}
		if err := baseState.UpdateDeployments(dsm.log, ds...); err != nil {
			return nil, errors.Wrapf(err, cluster)
		}
	}
	return baseState, nil
}

// WriteState implements StateManager on DispatchStateManager.
func (dsm *DispatchStateManager) WriteState(state *State, user User) error {
	deps, err := state.Deployments()
	if err != nil {
		return err
	}
	for cn, cm := range dsm.remotes {
		cds := deps.Filter(func(d *Deployment) bool {
			return d.ClusterName == cn
		})
		if err := cm.WriteCluster(cn, cds, user); err != nil {
			return errors.Wrapf(err, cn)
		}
	}
	return nil
}

// ReadCluster implements ClusterManager on DispatchStateManager.
func (dsm *DispatchStateManager) ReadCluster(clusterName string) (Deployments, error) {
	cm, ok := dsm.remotes[clusterName]
	if !ok {
		return Deployments{}, errors.Errorf("No cluster manager for %q", clusterName)
	}
	return cm.ReadCluster(clusterName)
}

// WriteCluster implements ClusterManager on DispatchStateManager.
func (dsm *DispatchStateManager) WriteCluster(clusterName string, deps Deployments, user User) error {
	cm, ok := dsm.remotes[clusterName]
	if !ok {
		return errors.Errorf("No cluster manager for %q", clusterName)
	}
	return cm.WriteCluster(clusterName, deps, user)
}
