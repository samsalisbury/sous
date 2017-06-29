package server

import (
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// A LiveGDM wraps a sous.Deployments and gets refreshed per server request
	LiveGDM struct {
		Etag string
		sous.Deployments
	}
)

// AddsPerRequest registers items into a SousGraph that need to be fresh per request
func AddsPerRequest(g restful.Injector) {
	g.Add(liveGDM)
	g.Add(getUser)
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
	etag, _ := state.GetEtag()
	return &LiveGDM{Etag: etag, Deployments: gdm.Deployments}, nil
}
