package singularity

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

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
	log.Println("Watching pending deployments for deploy ID:", pair.Post.SchedulerDID)

	counter := 0

WAIT_FOR_NOT_PENDING:

	pending, err := client.GetPendingDeploys()

	if err != nil {
		return nil, malformedResponse{"Getting pending deploys:" + err.Error()}
	}
	pds := make([]string, len(pending))
	for i, p := range pending {
		pds[i] = p.DeployMarker.DeployId
	}
	log.Println("Watching pending deployments for deploy ID:", pair.Post.SchedulerDID)

	log.Printf("Counter: %d - There are %d pending deploys: %s", counter, len(pending), strings.Join(pds, ", "))

	//HACK
	//poor man's wait, which seems to work since always, the item is in the queue, just the guid is off
	//we might wait a little extra since others might be in queue, but counter will prevent run away
	if len(pending) > 0 && counter < 600 {
		time.Sleep(2 * time.Second)
		counter = counter + 1
		goto WAIT_FOR_NOT_PENDING
	}

	/*
		for _, p := range pending {
			if p.DeployMarker.DeployId == pair.Post.SchedulerDID && counter < 600 {
				counter = counter + 1
				time.Sleep(2 * time.Second) //this is what OTPL does
				goto WAIT_FOR_NOT_PENDING
			}
		}
	*/
	j, err := json.Marshal(pending)
	if err != nil {
		log.Printf("ERROR: Last response from singularity pending deploys: %s", err)
	}
	log.Printf("Last response from singularity pending deploys: %s", string(j))

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
