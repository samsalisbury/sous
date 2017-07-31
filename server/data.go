package server

import (
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// NameData structs contain the pair of clustername to URL for data transfer
	NameData struct {
		ClusterName string
		URL         string
	}

	// ServerListData is the DTO for lists of servers
	ServerListData struct { // not actually a stutter - "server" means two different things.
		Servers []NameData
	}

	// GDMWrapper is the DTO wrapper for sous.Deployments
	GDMWrapper struct {
		Deployments []*sous.Deployment
	}

	// A LiveGDM wraps a sous.Deployments and gets refreshed per server request
	LiveGDM struct {
		Etag string
		sous.Deployments
	}

	// ClientUser is a local alias for sous.User
	ClientUser sous.User

	// StateManager is a DI adapter
	StateManager struct {
		sous.StateManager
	}
)

// EmptyReceiver implements Comparable on ServerListData
func (ld *ServerListData) EmptyReceiver() restful.Comparable {
	return &ServerListData{Servers: []NameData{}}
}

// VariancesFrom implements Comparable on ServerListData
func (ld *ServerListData) VariancesFrom(other restful.Comparable) restful.Variances {
	switch ol := other.(type) {
	default:
		return restful.Variances{"not a list of Deployments"}
	case *ServerListData:
		if len(ld.Servers) != len(ol.Servers) {
			return restful.Variances{"server list lengths differ"}
		}
		for _, l := range ld.Servers {
			var found *NameData
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
func (g *GDMWrapper) EmptyReceiver() restful.Comparable {
	return &GDMWrapper{Deployments: []*sous.Deployment{}}
}

// VariancesFrom implements Comparable on gdmWrapper
func (g *GDMWrapper) VariancesFrom(other restful.Comparable) restful.Variances {
	switch og := other.(type) {
	default:
		return restful.Variances{"Not a gdmWrapper"}
	case *GDMWrapper:
		return g.unwrap().VariancesFrom(og.unwrap())
	}
}

func (g *GDMWrapper) unwrap() *sous.Deployments {
	ds := sous.NewDeployments(g.Deployments...)
	return &ds
}
