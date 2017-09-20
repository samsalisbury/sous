package graph

import (
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
)

func newServerComponentLocator(ls LogSink, cfg LocalSousConfig, ins sous.Inserter, sm *ServerStateManager, rf *sous.ResolveFilter, ar *sous.AutoResolver) server.ComponentLocator {
	return server.ComponentLocator{
		LogSink:       ls.LogSink,
		Config:        cfg.Config,
		Inserter:      ins,
		StateManager:  sm.StateManager,
		ResolveFilter: rf,
		AutoResolver:  ar,
	}

}
