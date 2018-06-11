package graph

import (
	"fmt"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
	"github.com/samsalisbury/semv"
)

func newServerComponentLocator(
	ls LogSink,
	cfg LocalSousConfig,
	ins serverInserter,
	sm *ServerStateManager,
	cm *ServerClusterManager,
	rf *sous.ResolveFilter,
	ar *sous.AutoResolver,
	v semv.Version,
	qs *sous.R11nQueueSet,
) server.ComponentLocator {

	logging.Deliver(ls, logging.SousGenericV1, logging.DebugLevel, logging.GetCallerInfo(),
		logging.MessageField(fmt.Sprintf("Building CL: State manager: %T %[1]p", sm.StateManager)))

	var dm sous.DeploymentManager

	switch ldm := sm.StateManager.(type) {
	default:
		dm = sous.MakeDeploymentManager(sm.StateManager, ls)
	case sous.DeploymentManager:
		dm = ldm
	}
	return server.ComponentLocator{

		LogSink:           ls.LogSink,
		Config:            cfg.Config,
		Inserter:          ins.Inserter,
		StateManager:      sm.StateManager,
		ClusterManager:    cm.ClusterManager,
		DeploymentManager: dm,
		ResolveFilter:     rf,
		AutoResolver:      ar,
		Version:           v,
		QueueSet:          qs,
	}

}

// NewR11nQueueSet returns a new queue set configured to start processing r11ns
// immediately.
func NewR11nQueueSet(d sous.Deployer, r sous.Registry, rf *sous.ResolveFilter, sm *ServerStateManager) *sous.R11nQueueSet {
	sr := sm.StateManager
	return sous.NewR11nQueueSet(sous.R11nQueueStartWithHandler(
		func(qr *sous.QueuedR11n) sous.DiffResolution {
			qr.Rectification.Begin(d, r, rf, sr)
			return qr.Rectification.Wait()
		}))
}
