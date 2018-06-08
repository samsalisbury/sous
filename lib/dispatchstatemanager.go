package sous

import (
	"fmt"

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
func NewDispatchStateManager(
	localCluster string,
	clusters []string,
	local StateManager,
	remote ClusterManager,
	ls logging.LogSink,
) *DispatchStateManager {
	dsm := &DispatchStateManager{
		local:   local,
		remotes: map[string]ClusterManager{},
		log:     ls,
	}
	for _, n := range clusters {
		dsm.remotes[n] = remote
	}
	dsm.remotes[localCluster] = MakeClusterManager(local, ls)
	return dsm
}

// ReadState implements StateManager on DispatchStateManager.
func (dsm *DispatchStateManager) ReadState() (*State, error) {
	logging.DebugMsg(dsm.log, "DispatchStateManager ReadState")
	baseState, err := dsm.local.ReadState() // ReadState to get e.g. Defs
	if err != nil {
		return nil, errors.Wrapf(err, "base state")
	}
	for cluster, manager := range dsm.remotes {
		logging.DebugMsg(dsm.log, fmt.Sprintf("DispatchStateManager ReadState %q %T %[2]p", cluster, manager))
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
	logging.DebugMsg(dsm.log, "DispatchStateManager WriteState")
	deps, err := state.Deployments()
	if err != nil {
		return err
	}
	for cn, cm := range dsm.remotes {
		logging.Debug(dsm.log, fmt.Sprintf("DispatchStateManager WriteState %q %T", cn, cm))
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
	logging.DebugMsg(dsm.log, fmt.Sprintf("DispatchStateManager ReadCluster %q %T", clusterName, cm))
	if !ok {
		return Deployments{}, errors.Errorf("No cluster manager for %q", clusterName)
	}
	return cm.ReadCluster(clusterName)
}

// WriteCluster implements ClusterManager on DispatchStateManager.
func (dsm *DispatchStateManager) WriteCluster(clusterName string, deps Deployments, user User) error {
	cm, ok := dsm.remotes[clusterName]
	logging.DebugMsg(dsm.log, fmt.Sprintf("DispatchStateManager WriteCluster %q %T", clusterName, cm))
	if !ok {
		return errors.Errorf("No cluster manager for %q", clusterName)
	}
	return cm.WriteCluster(clusterName, deps, user)
}
