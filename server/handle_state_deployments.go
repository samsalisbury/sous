package server

import (
	"encoding/json"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// A StateDeploymentResource provides for the /state/deployments resource family
	StateDeploymentResource struct {
		loc ComponentLocator
		userExtractor
	}

	// A GETStateDeployments is the exchanger for GET /state/deployments
	GETStateDeployments struct {
		cluster     sous.ClusterManager
		clusterName string
	}

	// A PUTStateDeployments is the exchanger for PUT /state/deployments
	PUTStateDeployments struct {
		cluster     sous.ClusterManager
		clusterName string
		req         *http.Request
		User        ClientUser
	}
)

func newStateDeploymentResource(loc ComponentLocator) *StateDeploymentResource {
	return &StateDeploymentResource{loc: loc}
}

// Get implements restful.Getable on StateDeployments
func (res *StateDeploymentResource) Get(http.ResponseWriter, *http.Request, httprouter.Params) restful.Exchanger {
	spew.Dump(res.loc.ResolveFilter)
	return &GETStateDeployments{
		cluster:     res.loc.ClusterManager,
		clusterName: res.loc.ResolveFilter.Cluster.ValueOr("no-cluster"),
	}
}

// Put implements restful.Putable on StateDeployments
func (res *StateDeploymentResource) Put(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &PUTStateDeployments{
		cluster:     res.loc.ClusterManager,
		clusterName: res.loc.ResolveFilter.Cluster.ValueOr("no-cluster"),
		req:         req,
		User:        res.GetUser(req),
	}
}

// Exchange implements restful.Exchanger on GETStateDeployments
func (gsd *GETStateDeployments) Exchange() (interface{}, int) {
	data := GDMWrapper{Deployments: []*sous.Deployment{}}
	spew.Dump(gsd)
	deps, err := gsd.cluster.ReadCluster(gsd.clusterName)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	for _, d := range deps.Snapshot() {
		data.Deployments = append(data.Deployments, d)
	}

	return data, http.StatusOK
}

// Exchange implements Exchanger on PUTStateDeployments
func (psd *PUTStateDeployments) Exchange() (interface{}, int) {
	data := GDMWrapper{}
	dec := json.NewDecoder(psd.req.Body)
	err := dec.Decode(&data)
	if err != nil {
		return err, http.StatusBadRequest
	}

	deps := sous.NewDeployments(data.Deployments...)

	err = psd.cluster.WriteCluster(psd.clusterName, deps, sous.User(psd.User))
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusAccepted
}
