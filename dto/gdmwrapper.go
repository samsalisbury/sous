package dto

import (
	"crypto/sha512"
	"encoding/base64"
	"net/http"
	"sort"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

// GDMWrapper is the DTO wrapper for sous.Deployments
type GDMWrapper struct {
	Deployments []*sous.Deployment
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
// n.b. that Etags are generated automatically for most restful bodies
// GDMWrapper is unique because any *set* of Deployments is equivalent, regardless of their order.
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
