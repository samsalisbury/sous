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
func (sc *deployer) GetRunningDeployment(singMap map[string]string) (deps sous.Deployments, err error) {
	retries := make(retryCounter)
	errCh := make(chan error)
	deps = make(sous.Deployments, 0)
	sings := make(map[string]struct{})
	reqCh := make(chan SingReq, len(singMap)*ReqsPerServer)
	depCh := make(chan *sous.Deployment, ReqsPerServer)

	defer close(depCh)
	// XXX The intention here was to use something like the gotools context to
	// manage NW cancellation
	//defer sc.rectClient.Cancel()

	var singWait, depWait sync.WaitGroup

	singWait.Add(len(singMap))
	for _, url := range singMap {
		if _, ok := sings[url]; ok {
			continue
		}
		//sing.Debug = true
		sings[url] = struct{}{}
		client := sc.buildSingClient(url)
		go singPipeline(url, client, &depWait, &singWait, reqCh, errCh)
	}

	go depPipeline(sc.Client, singMap, MaxAssemblers, reqCh, depCh, errCh)

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
			deps = append(deps, dep)
			Log.Debug.Printf("Deployment #%d: %+v", len(deps), dep)
			depWait.Done()
		case err = <-errCh:
			if _, ok := err.(malformedResponse); ok {
				Log.Notice.Print(err)
				depWait.Done()
			} else {
				retried := retries.maybe(err, reqCh)
				if !retried {
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
		Log.Debug.Print(err)
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
		Log.Vomit.Print(err)
		errs <- errors.Wrap(err, "getting request list")
		return
	}
	for _, r := range rs {
		Log.Vomit.Print("Req: ", r)
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
	nicks map[string]string,
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

			dep, err := assembleDeployment(cl, nicks, req)

			if err != nil {
				errCh <- errors.Wrap(err, "assembly problem")
			} else {
				depCh <- dep
			}
		}(cl, req)
	}
}

func assembleDeployment(cl rectificationClient, nicks map[string]string, req SingReq) (*sous.Deployment, error) {
	Log.Vomit.Print("Assembling from: ", req)
	tgt, err := BuildDeployment(cl, nicks, req)
	if err != nil {
		Log.Vomit.Print(err)
		return nil, errors.Wrap(err, "Building deployment")
	}

	Log.Vomit.Printf("Collected deployment: %v", tgt)
	return &tgt, nil
}
