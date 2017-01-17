package server

import (
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
)

type (
	// A LiveGDM wraps a sous.Deployments and gets refreshed per server request
	LiveGDM struct {
		sous.Deployments
	}
)

// AddsPerRequest registers items into a SousGraph that need to be fresh per request
func AddsPerRequest(g Injector) {
	g.Add(liveGDM)
}

func liveGDM(sr graph.StateReader) (*LiveGDM, error) {
	state, err := graph.NewCurrentState(sr)
	if err != nil {
		return nil, err
	}
	gdm, err := graph.NewCurrentGDM(state)
	if err != nil {
		return nil, err
	}
	return &LiveGDM{gdm.Deployments}, nil
}
