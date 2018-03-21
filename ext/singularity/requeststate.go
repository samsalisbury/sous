package singularity

import (
	"fmt"

	sous "github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

// Status implements sous.Deployer on deployer.
func (r *deployer) Status(reg sous.Registry, clusters sous.Clusters, pair *sous.DeployablePair) (*sous.DeployState, error) {
	var url string

	reqID, err := r.getRequestID(pair.Post)
	if err != nil {
		return nil, err
	}

	clusterName := pair.Post.Deployment.ClusterName
	if cluster, has := clusters[clusterName]; has {
		url = cluster.BaseURL
	} else {
		return nil, errors.Errorf("No cluster found for %q. Known are: %q.", clusterName, clusters.Names())
	}

	client := r.buildSingClient(url)

WAIT_FOR_NOT_PENDING:

	pending, err := client.GetPendingDeploys()
	//if pair.Post.SchedulerDID == "" {
	//	goto SKIP_PENDING_CHECK
	//}
	if err != nil {
		return nil, malformedResponse{"Getting pending deploys:" + err.Error()}
	}
	for _, p := range pending {
		if p.DeployMarker.DeployId == pair.Post.SchedulerDID {
			goto WAIT_FOR_NOT_PENDING
		}
	}

	//SKIP_PENDING_CHECK:

	reqParent, err := client.GetRequest(reqID, false) //don't use the web cache
	if err != nil {
		return nil, err
	}

	singReq := SingReq{
		SourceURL: url,
		Sing:      client,
		ReqParent: reqParent,
	}

	tgt, err := BuildDeployment(reg, clusters, singReq, r.log)

	tgt.SchedulerURL = fmt.Sprintf("http://%s/request/%s", url, reqID)

	return &tgt, errors.Wrapf(err, "getting request state")
}

func (r *deployer) getRequestID(d *sous.Deployable) (string, error) {
	// TODO: add a cache of known Deployables to their Requests (and current state...)
	return computeRequestID(d)
}
