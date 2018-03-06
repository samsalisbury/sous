package server

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sort"

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

	// ClientUser is a local alias for sous.User
	ClientUser sous.User

	// StateManager is a DI adapter
	StateManager struct {
		sous.StateManager
	}

	// DeploymentQueuesResponse is used by the Deployment queue handler
	DeploymentQueuesResponse struct {
		Queues map[string]QueueDesc
	}

	// QueueDesc describes the queue related to a Deployment
	QueueDesc struct {
		sous.DeploymentID
		Length int
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

// AddHeaders implements HeaderAdder on GDMWrapper
// GDMWrappers add an Etag to the response
func (g GDMWrapper) AddHeaders(headers http.Header) {
	headers.Add("Etag", g.etag())
}

// Etag returns a string suitable for use in an Etag header for this data type.
func (g GDMWrapper) etag() string {
	deps := make([]*sous.Deployment, 0, len(g.Deployments))
	copy(deps, g.Deployments)
	sort.Slice(deps, func(i, j int) bool { return deps[i].ID().String() < deps[j].ID().String() })

	hash := sha512.New()
	for _, dep := range deps {
		hash.Write([]byte(dep.String()))
	}

	return "w/" + base64.URLEncoding.EncodeToString(hash.Sum(nil))
}

func (g *GDMWrapper) unwrap() *sous.Deployments {
	ds := sous.NewDeployments(g.Deployments...)
	return &ds
}

// SingleDeploymentBody is the response struct returned from handlers
// of HTTP methods of a SingleDeploymentResource.
type SingleDeploymentBody struct {
	Meta           ResponseMeta
	DeploySpec     sous.DeploySpec
	ManifestHeader sous.Manifest
}

// Etag returns a string suitable for use in an Etag header for this data type.
func (b *SingleDeploymentBody) etag() string {
	hash := sha512.New()
	ds, err := json.Marshal(b.DeploySpec)
	if err != nil {
		panic("unmarshallable SingleDeploymentBody.DeploySpec")
	}
	mh, err := json.Marshal(b.ManifestHeader)
	if err != nil {
		panic("unmarshallable SingleDeploymentBody.ManifestHeader")
	}

	hash.Write(ds)
	hash.Write(mh)
	return "w/" + base64.URLEncoding.EncodeToString(hash.Sum(nil))
}
