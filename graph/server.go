package graph

import (
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/logging"
)

func newServerComponentLocator(ls logging.LogSet, cfg LocalSousConfig, ins sous.Inserter, sm *ServerStateManager, rf *sous.ResolveFilter, ar *sous.AutoResolver) server.ComponentLocator {
	return server.ComponentLocator{
		LogSet:        ls,
		Config:        cfg.Config,
		Inserter:      ins,
		StateManager:  sm.StateManager,
		ResolveFilter: rf,
		AutoResolver:  ar,
	}

}
