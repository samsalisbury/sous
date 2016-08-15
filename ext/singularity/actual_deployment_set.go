package singularity

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

// ReqsPerServer limits the number of simultaneous number of requests made
// against a single Singularity server
const (
	ReqsPerServer = 10
	MaxAssemblers = 100
)

type (
	sDeploy    *dtos.SingularityDeploy
	sRequest   *dtos.SingularityRequest
	sDepMarker *dtos.SingularityDeployMarker

	// SingReq captures a request made to singularity with its initial response
	SingReq struct {
		SourceURL string
		Sing      *singularity.Client
		ReqParent *dtos.SingularityRequestParent
	}

	retryCounter map[string]uint
)

// GetRunningDeployment collects data from the Singularity clusters and
// returns a list of actual deployments
func (sc *deployer) GetRunningDeployments(clusters sous.Clusters) (deps sous.Deployments, err error) {
	retries := make(retryCounter)
	errCh := make(chan error)
	deps = sous.NewDeployments()
	sings := make(map[string]struct{})
	reqCh := make(chan SingReq, len(clusters)*ReqsPerServer)
	depCh := make(chan *sous.Deployment, ReqsPerServer)

	defer close(depCh)
	// XXX The intention here was to use something like the gotools context to
	// manage NW cancellation
	//defer sc.rectClient.Cancel()

	var singWait, depWait sync.WaitGroup

	singWait.Add(len(clusters))
	for _, url := range clusters {
		url := url.BaseURL
		if _, ok := sings[url]; ok {
			continue
		}
		//sing.Debug = true
		sings[url] = struct{}{}
		client := sc.buildSingClient(url)
		go singPipeline(url, client, &depWait, &singWait, reqCh, errCh)
	}

	go depPipeline(sc.Client, clusters, MaxAssemblers, reqCh, depCh, errCh)

	go func() {
		catchAndSend("closing up", errCh)
		singWait.Wait()
		depWait.Wait()

		close(reqCh)
		close(errCh)
	}()

	for {
		select {
		case dep := <-depCh:
			deps.Add(dep)
			Log.Debug.Printf("Deployment #%d: %+v", deps.Len(), dep)
			depWait.Done()
		case err = <-errCh:
			if isMalformed(err) {
				Log.Debug.Print(err)
				depWait.Done()
			} else {
				retried := retries.maybe(err, reqCh)
				if !retried {
					Log.Notice.Print("Cannot retry: ", err)
					return
				}
			}
		}
	}
}

const retryLimit = 3

func (rc retryCounter) maybe(err error, reqCh chan SingReq) bool {
	rt, ok := err.(*canRetryRequest)
	if !ok {
		return false
	}

	Log.Debug.Printf("%T err = %+v\n", err, err)
	count, ok := rc[rt.name()]
	if !ok {
		count = 0
	}
	if count > retryLimit {
		return false
	}

	rc[rt.name()] = count + 1
	go func() {
		defer catchAll("retrying: " + rt.req.SourceURL)
		time.Sleep(time.Millisecond * 50)
		reqCh <- rt.req
	}()

	return true
}

func catchAll(from string) {
	if err := recover(); err != nil {
		Log.Warn.Printf("Recovering from %s where we received %v", from, err)
	}
}

func dontrecover() error {
	return nil
}

func catchAndSend(from string, errs chan error) {
	defer catchAll(from)
	if err := dontrecover(); err != nil {
		Log.Debug.Printf("from = %s err = %+v\n", from, err)
		Log.Debug.Printf("debug.Stack() = %+v\n", string(debug.Stack()))
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

func logFDs(when string) {
	defer func() { recover() }() // this is just diagnostic
	pid := os.Getpid()
	fdDir, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		Log.Vomit.Print(err)
		return
	}
	for _, f := range fdDir {
		n, e := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, f.Name()))
		if e != nil {
			n = f.Name()
		}

		Log.Vomit.Printf("%s: %s", f.Mode(), n)
	}

	Log.Debug.Printf("%s: %d", when, len(fdDir))
}

func singPipeline(
	url string,
	client *singularity.Client,
	dw, wg *sync.WaitGroup,
	reqs chan SingReq,
	errs chan error,
) {
	defer wg.Done()
	defer catchAndSend(fmt.Sprintf("get requests: %s", url), errs)
	rs, err := getRequestsFromSingularity(url, client)
	if err != nil {
		Log.Vomit.Print(err) //XXX connection reset by peer should be retried
		errs <- errors.Wrap(err, "getting request list")
		return
	}
	for _, r := range rs {
		Log.Vomit.Printf("Req: %s %s", r.SourceURL, reqID(r.ReqParent))
		dw.Add(1)
		reqs <- r
	}
}

func getRequestsFromSingularity(url string, client *singularity.Client) ([]SingReq, error) {
	logFDs("before getRequestsFromSingularity")
	defer logFDs("after getRequestsFromSingularity")
	singRequests, err := client.GetRequests()
	if err != nil {
		return nil, errors.Wrap(err, "getting request")
	}

	reqs := make([]SingReq, 0, len(singRequests))
	for _, sr := range singRequests {
		reqs = append(reqs, SingReq{url, client, sr})
	}

	return reqs, nil
}

func depPipeline(
	cl rectificationClient,
	clusters sous.Clusters,
	poolCount int,
	reqCh chan SingReq,
	depCh chan *sous.Deployment,
	errCh chan error,
) {
	defer catchAndSend("dependency building", errCh)
	poolLimit := make(chan struct{}, poolCount)
	for req := range reqCh {
		go func(cl rectificationClient, req SingReq) {
			defer catchAndSend(fmt.Sprintf("dep from req %s", req.SourceURL), errCh)

			poolLimit <- struct{}{}
			defer func() { <-poolLimit }()

			dep, err := assembleDeployment(cl, clusters, req)

			if err != nil {
				errCh <- errors.Wrap(err, "assembly problem")
			} else {
				depCh <- dep
			}
		}(cl, req)
	}
}

func assembleDeployment(cl rectificationClient, clusters sous.Clusters, req SingReq) (*sous.Deployment, error) {
	Log.Vomit.Printf("Assembling from: %s %s", req.SourceURL, reqID(req.ReqParent))
	tgt, err := BuildDeployment(cl, clusters, req)
	if err != nil {
		return nil, errors.Wrap(err, "Building deployment")
	}

	Log.Vomit.Printf("Collected deployment: %v", tgt)
	return &tgt, nil
}
