package singularity

import (
	"fmt"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
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

	if pair.UUID == uuid.Nil {
		pair.UUID = uuid.NewV4()
	}
	depID := computeDeployIDFromUUID(pair.Post, pair.UUID)

	client := r.buildSingClient(url)

	messages.ReportLogFieldsMessageToConsole(
		fmt.Sprintf("Watching pending deployments for deploy ID: %s", depID),
		logging.ExtraDebug1Level, r.log, depID)

	reqParent, err := client.GetRequest(reqID, false) //don't use the web cache
	if err != nil {
		return nil, errors.Wrapf(err, "getting request")
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
