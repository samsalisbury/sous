package sous

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/opentable/singularity"
	"github.com/opentable/singularity/dtos"
	"github.com/opentable/sous/util/docker_registry"
)

// ReqsPerServer limits the number of simultaneous number of requests made
// against a single Singularity server
const ReqsPerServer = 10

type (
	sDeploy    *dtos.SingularityDeploy
	sRequest   *dtos.SingularityRequest
	sDepMarker *dtos.SingularityDeployMarker

	singReq struct {
		sourceURL string
		sing      *singularity.Client
		reqParent *dtos.SingularityRequestParent
	}

	retryCounter map[string]uint
)

// GetRunningDeploymentSet collects data from the Singularity clusters and
// returns a list of actual deployments
func GetRunningDeploymentSet(singUrls []string) (deps Deployments, err error) {
	retries := make(retryCounter)
	errCh := make(chan error)
	deps = make(Deployments, 0)
	sings := make(map[string]*singularity.Client)

	reqCh := make(chan singReq, len(singUrls)*ReqsPerServer)
	depCh := make(chan *Deployment, ReqsPerServer)
	defer close(depCh)

	regClient := docker_registry.NewClient()
	regClient.BecomeFoolishlyTrusting()
	defer regClient.Cancel()

	var singWait, depWait sync.WaitGroup

	singWait.Add(len(singUrls))
	for _, url := range singUrls {
		sing := singularity.NewClient(url)
		sings[url] = sing
		go singPipeline(sing, &depWait, &singWait, reqCh, errCh)
	}

	go depPipeline(regClient, reqCh, depCh, errCh)

	go func() {
		catchAndSend("closing up", errCh)
		singWait.Wait()
		close(reqCh)

		depWait.Wait()
		close(errCh)
	}()

	for {
		select {
		case dep := <-depCh:
			deps = append(deps, dep)
			depWait.Done()
		case err = <-errCh:
			retried := retries.maybe(err, reqCh)
			if !retried {
				return
			}
		}
	}
}

const retryLimit = 3

func (rc retryCounter) maybe(err error, reqCh chan singReq) bool {
	rt, ok := err.(*canRetryRequest)
	if !ok {
		return false
	}

	count, ok := rc[rt.name()]
	if !ok {
		count = 0
	}
	if count > retryLimit {
		return false
	}

	rc[rt.name()] = count + 1
	go func() {
		time.Sleep(time.Millisecond * 50)
		reqCh <- rt.req
	}()

	return true
}

func catchAll(from string) {
	if err := recover(); err != nil {
		log.Printf("Recovering from %s where we received %v", from, err)
	}
}

func catchAndSend(from string, errs chan error) {
	defer catchAll(from)
	if err := recover(); err != nil {
		log.Printf("from = %s err = %+v\n", from, err)
		log.Printf("debug.Stack() = %+v\n", string(debug.Stack()))
		switch err := err.(type) {
		default:
			if err != nil {
				errs <- fmt.Errorf("Panicked with not-error: %v", err)
			}
		case error:
			errs <- fmt.Errorf("at %s: %v", from, err)
		}
	}
}

func singPipeline(client *singularity.Client, dw, wg *sync.WaitGroup, reqs chan singReq, errs chan error) {
	defer wg.Done()
	defer catchAndSend(fmt.Sprintf("get requests: %s", client), errs)
	rs, err := getRequestsFromSingularity(client)
	if err != nil {
		errs <- err
		return
	}
	for _, r := range rs {
		dw.Add(1)
		reqs <- r
	}
}

func getRequestsFromSingularity(client *singularity.Client) ([]singReq, error) {
	singRequests, err := client.GetRequests()
	if err != nil {
		return nil, err
	}

	reqs := make([]singReq, 0, len(singRequests))
	for _, sr := range singRequests {
		reqs = append(reqs, singReq{client.BaseUrl, client, sr})
	}

	return reqs, nil
}

func depPipeline(cl docker_registry.Client, reqCh chan singReq, depCh chan *Deployment, errCh chan error) {
	defer catchAndSend("dependency building", errCh)
	for req := range reqCh {
		go func(cl docker_registry.Client, req singReq) {
			defer catchAndSend(fmt.Sprintf("dep from req %s", req.sourceURL), errCh)

			dep, err := assembleDeployment(cl, req)

			if err != nil {
				errCh <- err
			}

			depCh <- dep
		}(cl, req)
	}
}

func assembleDeployment(cl docker_registry.Client, req singReq) (*Deployment, error) {
	uc := newDeploymentBuilder(cl, req)
	err := uc.completeConstruction()
	if err != nil {
		return nil, err
	}

	return &uc.target, nil
}
