package graph

import (
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/samsalisbury/semv"
)

func newServerComponentLocator(ls LogSink, cfg LocalSousConfig, ins sous.Inserter, sm *ServerStateManager, rf *sous.ResolveFilter, ar *sous.AutoResolver, v semv.Version) server.ComponentLocator {
	return server.ComponentLocator{
		LogSink:       ls.LogSink,
		Config:        cfg.Config,
		Inserter:      ins,
		StateManager:  sm.StateManager,
		ResolveFilter: rf,
		AutoResolver:  ar,
		Version:       v,
	}

}
