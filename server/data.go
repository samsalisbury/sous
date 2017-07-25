package server

import (
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	server struct {
		ClusterName string
		URL         string
	}

	serverListData struct {
		Servers []server
	}

	gdmWrapper struct {
		Deployments []*sous.Deployment
	}

	// A LiveGDM wraps a sous.Deployments and gets refreshed per server request
	LiveGDM struct {
		Etag string
		sous.Deployments
	}
)

func (ld *serverListData) EmptyReceiver() restful.Comparable {
	return &serverListData{Servers: []server{}}
}

func (ld *serverListData) VariancesFrom(other restful.Comparable) restful.Variances {
	switch ol := other.(type) {
	default:
		return restful.Variances{"not a list of Deployments"}
	case *serverListData:
		if len(ld.Servers) != len(ol.Servers) {
			return restful.Variances{"server list lengths differ"}
		}
		for _, l := range ld.Servers {
			var found *server
			for _, r := range ol.Servers {
				if l.ClusterName == r.ClusterName && l.URL == r.URL {
					found = &r
					break
				}
			}
			if found == nil {
				return restful.Variances{"No match found for " + l.ClusterName}
			}
		}
		return restful.Variances{}
	}
}

// EmptyReceiver implements Comparable on gdmWrapper
func (g *gdmWrapper) EmptyReceiver() restful.Comparable {
	return &gdmWrapper{Deployments: []*sous.Deployment{}}
}

// VariancesFrom implements Comparable on gdmWrapper
func (g *gdmWrapper) VariancesFrom(other restful.Comparable) restful.Variances {
	switch og := other.(type) {
	default:
		return restful.Variances{"Not a gdmWrapper"}
	case *gdmWrapper:
		return g.unwrap().VariancesFrom(og.unwrap())
	}
}

func (g *gdmWrapper) unwrap() *sous.Deployments {
	ds := sous.NewDeployments(g.Deployments...)
	return &ds
}
