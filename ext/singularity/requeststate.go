package singularity

import (
	"fmt"
	"strings"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
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

	depID := computeDeployID(pair.Post)
	client := r.buildSingClient(url)

	messages.ReportLogFieldsMessageToConsole(
		fmt.Sprintf("Watching pending deployments for deploy ID: %s", pair.Post.SchedulerDID),
		logging.ExtraDebug1Level, r.log, pair.Post.SchedulerDID)

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
	if !tgt.Status.Failed() {
		status, err := r.checkPendingList(pair.Post, client)
		if err != nil {
			return nil, errors.Wrapf(err, "getting pending state")
		}
		tgt.Status = status
	}

	return &tgt, errors.Wrapf(err, "getting request state")
}

func (r *deployer) getRequestID(d *sous.Deployable) (string, error) {
	// TODO: add a cache of known Deployables to their Requests (and current state...)
	return computeRequestID(d)
}

func (r *deployer) checkPendingList(d *sous.Deployable, client singClient) (sous.DeployStatus, error) {
	depID := computeDeployID(d)

	pending, err := client.GetPendingDeploys()

	if err != nil {
		return sous.DeployStatusAny, malformedResponse{"Getting pending deploys:" + err.Error()}
	}
	pds := make([]string, len(pending))
	for i, p := range pending {
		pds[i] = p.DeployMarker.DeployId
	}

	messages.ReportLogFieldsMessageToConsole(
		fmt.Sprintf("Watching pending deployments for deploy ID: %s", depID),
		logging.ExtraDebug1Level, r.log, depID)
	messages.ReportLogFieldsMessageToConsole(
		fmt.Sprintf("There are %d pending deploys: %s", len(pending), strings.Join(pds, ", ")),
		logging.ExtraDebug1Level, r.log, depID)

	for _, p := range pending {
		if p.DeployMarker.DeployId == depID {
			return sous.DeployStatusPending, nil
		}
	}
	return sous.DeployStatusActive, nil
}
