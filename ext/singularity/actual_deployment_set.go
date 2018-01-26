package singularity

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

const (
	// MaxAssemblers is the maximum number of simultaneous deployment
	// assemblers.
	MaxAssemblers = 100
)

type (
	sHistory   *dtos.SingularityDeployHistory
	sDeploy    *dtos.SingularityDeploy
	sRequest   *dtos.SingularityRequest
	sDepMarker *dtos.SingularityDeployMarker

	// A SingClient provides the interface needed to assemble a deployment.
	SingClient interface {
		GetDeploy(requestID string, deployID string) (*dtos.SingularityDeployHistory, error)
		GetDeploys(requestID string, count, page int32) (dtos.SingularityDeployHistoryList, error)
	}

	// SingReq captures a request made to singularity with its initial response
	SingReq struct {
		SourceURL string
		Sing      SingClient
		ReqParent *dtos.SingularityRequestParent
	}

	retryCounter struct {
		count map[string]uint
		log   logging.LogSink
	}
)

// RunningDeployments collects data from the Singularity clusters and
// returns a list of actual deployments.
func (sc *deployer) RunningDeployments(reg sous.Registry, clusters sous.Clusters) (sous.DeployStates, error) {
	var deps sous.DeployStates
	retries := retryCounter{
		count: map[string]uint{},
		log:   sc.log,
	}
	errCh := make(chan error)
	deps = sous.NewDeployStates()
	sings := make(map[string]struct{})
	reqCh := make(chan SingReq, len(clusters)*sc.ReqsPerServer)
	depCh := make(chan *sous.DeployState, sc.ReqsPerServer)

	defer close(depCh)
	// XXX The intention here was to use something like the gotools context to
	// manage NW cancellation
	//defer sc.rectClient.Cancel()

	var depAssWait, singWait, depWait sync.WaitGroup

	sc.log.Vomitf("Setting up to wait for %d clusters", len(clusters))
	singWait.Add(len(clusters))
	for _, url := range clusters {
		url := url.BaseURL
		if _, ok := sings[url]; ok {
			singWait.Done()
			continue
		}
		//sing.Debug = true
		sings[url] = struct{}{}
		client := sc.buildSingClient(url)
		go sc.singPipeline(reg, url, client, &depWait, &singWait, reqCh, errCh, clusters)
	}

	go sc.depPipeline(reg, clusters, MaxAssemblers, &depAssWait, reqCh, depCh, errCh)

	go func() {
		defer catchAndSend("closing channels", errCh, sc.log)

		singWait.Wait()
		sc.log.Debugf("All singularities polled for requests")

		depWait.Wait()
		sc.log.Debugf("All deploys processed")

		depAssWait.Wait()
		sc.log.Debugf("All deployments assembled")

		close(reqCh)
		sc.log.Debugf("Closed reqCh")
		close(errCh)
		sc.log.Debugf("Closed errCh")
	}()

	for {
		select {
		case dep := <-depCh:
			deps.Add(dep)
			sc.log.Debugf("Deployment #%d: %+v", deps.Len(), dep)
			depWait.Done()
		case err, cont := <-errCh:
			if !cont {
				sc.log.Debugf("Errors channel closed. Finishing up.")
				return deps, nil
			}
			if isMalformed(sc.log, err) || ignorableDeploy(err) {
				sc.log.Debugf("\n", err)
				depWait.Done()
			} else {
				retryable := retries.maybe(err, reqCh)
				if !retryable {
					sc.log.Warnf("Cannot retry: %v. Exiting", err)
					return deps, err
				}
			}
		}
	}
}

const retryLimit = 3

func (rc retryCounter) maybe(err error, reqCh chan SingReq) bool {
	rt, ok := errors.Cause(err).(*canRetryRequest)
	if !ok {
		return false
	}

	rc.log.Debugf("%T err = %+v\n", errors.Cause(err), errors.Cause(err))
	count, ok := rc.count[rt.name()]
	if !ok {
		count = 0
	}
	if count > retryLimit {
		return false
	}

	rc.count[rt.name()] = count + 1
	go func() {
		defer catchAll("retrying: "+rt.req.SourceURL, rc.log)
		time.Sleep(time.Millisecond * 50)
		reqCh <- rt.req
	}()

	return true
}

func catchAll(from string, log logging.LogSink) {
	if err := recover(); err != nil {
		log.Warnf("Recovering from %s where we received %v", from, err)
	}
}

func dontrecover() error {
	return nil
}

func catchAndSend(from string, errs chan error, log logging.LogSink) {
	defer catchAll(from, log)
	if err := recover(); err != nil {
		log.Debugf("from = %s err = %+v\n", from, err)
		log.Debugf("debug.Stack() = %+v\n", string(debug.Stack()))
		switch err := err.(type) {
		default:
			if err != nil {
				errs <- fmt.Errorf("%s: Panicked with not-error: %v", from, err)
			}
		case error:
			errs <- errors.Wrapf(err, from)
		}
	}
}

func (sc *deployer) singPipeline(
	reg sous.Registry,
	url string,
	client *singularity.Client,
	dw, wg *sync.WaitGroup,
	reqs chan SingReq,
	errs chan error,
	//	clusters []string,
	clusters sous.Clusters,
) {
	sc.log.Vomitf("Starting cluster at %s", url)
	defer func() { sc.log.Vomitf("Completed cluster at %s", url) }()
	defer wg.Done()
	defer catchAndSend(fmt.Sprintf("get requests: %s", url), errs, sc.log)
	srp, err := sc.getSingularityRequestParents(client)
	if err != nil {
		sc.log.Vomitf("%v", err) //XXX connection reset by peer should be retried
		errs <- errors.Wrap(err, "getting request list")
		return
	}

	rs := convertSingularityRequestParentsToSingReqs(url, client, srp)

	for _, r := range rs {
		sc.log.Vomitf("Req: %s %s %d", r.SourceURL, reqID(r.ReqParent), r.ReqParent.Request.Instances)
		dw.Add(1)
		reqs <- r
	}
}

func (sc *deployer) getSingularityRequestParents(client *singularity.Client) ([]*dtos.SingularityRequestParent, error) {
	singRequests, err := client.GetRequests(false) // = don't use the 30 second cache

	return singRequests, errors.Wrap(err, "getting request")
}

func convertSingularityRequestParentsToSingReqs(url string, client *singularity.Client, srp []*dtos.SingularityRequestParent) []SingReq {
	reqs := make([]SingReq, 0, len(srp))

	for _, sr := range srp {
		reqs = append(reqs, SingReq{url, client, sr})
	}
	return reqs
}

func (sc *deployer) depPipeline(
	reg sous.Registry,
	clusters sous.Clusters,
	poolCount int,
	depAssWait *sync.WaitGroup,
	reqCh chan SingReq,
	depCh chan *sous.DeployState,
	errCh chan error,
) {
	defer catchAndSend("dependency building", errCh, sc.log)
	poolLimit := make(chan struct{}, poolCount)
	for req := range reqCh {
		depAssWait.Add(1)
		sc.log.Vomitf("starting assembling for %q", reqID(req.ReqParent))
		go func(req SingReq) {
			defer depAssWait.Done()
			defer catchAndSend(fmt.Sprintf("dep from req %s", req.SourceURL), errCh, sc.log)
			poolLimit <- struct{}{}
			defer func() {
				sc.log.Vomitf("finished assembling for %q", reqID(req.ReqParent))
				<-poolLimit
			}()

			dep, err := sc.assembleDeployState(reg, clusters, req)

			if err != nil {
				errCh <- errors.Wrap(err, "assembly problem")
			} else {
				depCh <- dep
			}
		}(req)
	}
}

func (sc *deployer) assembleDeployState(reg sous.Registry, clusters sous.Clusters, req SingReq) (*sous.DeployState, error) {
	sc.log.Vomitf("Assembling from: %s %s", req.SourceURL, reqID(req.ReqParent))
	tgt, err := BuildDeployment(reg, clusters, req, sc.log)
	sc.log.Vomitf("Collected deployment: %#v", tgt)
	return &tgt, errors.Wrap(err, "Building deployment")
}
