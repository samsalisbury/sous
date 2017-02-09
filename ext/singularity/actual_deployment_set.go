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
// against a single Singularity server.
const (
	ReqsPerServer = 10
	MaxAssemblers = 100
)

type (
	sDeploy    *dtos.SingularityDeploy
	sRequest   *dtos.SingularityRequest
	sDepMarker *dtos.SingularityDeployMarker

	// Request captures a request made to singularity with its initial response.
	Request struct {
		URL           string
		Client        *singularity.Client
		RequestParent *dtos.SingularityRequestParent
	}

	retryCounter map[string]uint
)

// RunningDeployments collects data from the Singularity clusters and
// returns a list of actual deployments.
func (sc *deployer) RunningDeployments(reg sous.Registry, clusters sous.Clusters) (deps sous.DeployStates, err error) {
	retries := make(retryCounter)
	errCh := make(chan error)
	deps = sous.NewDeployStates()
	sings := make(map[string]struct{})
	reqCh := make(chan Request, len(clusters)*ReqsPerServer)
	depCh := make(chan *sous.DeployState, ReqsPerServer)

	defer close(depCh)
	// XXX The intention here was to use something like the gotools context to
	// manage NW cancellation
	//defer sc.rectClient.Cancel()

	var singWait, depWait sync.WaitGroup

	Log.Vomit.Printf("Setting up to wait for %d clusters", len(clusters))
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
		go singPipeline(url, client, &depWait, &singWait, reqCh, errCh, clusters.Names())
	}

	go depPipeline(sc.Client, reg, clusters, MaxAssemblers, reqCh, depCh, errCh)

	go func() {
		catchAndSend("closing up", errCh)
		singWait.Wait()
		Log.Debug.Println("All singularities polled for requests")
		depWait.Wait()
		Log.Debug.Println("All deploys processed")

		close(reqCh)
		close(errCh)
	}()

	for {
		var cont bool
		select {
		case dep := <-depCh:
			deps.Add(dep)
			Log.Debug.Printf("Deployment #%d: %+v", deps.Len(), dep)
			depWait.Done()
		case err, cont = <-errCh:
			if !cont {
				Log.Debug.Printf("Errors channel closed. Finishing up.")
				return
			}
			if isMalformed(err) {
				Log.Debug.Print(err)
				depWait.Done()
			} else {
				retried := retries.maybe(err, reqCh)
				if !retried {
					Log.Notice.Printf("Cannot retry: %v. Exiting", err)
					return
				}
			}
		}
	}
}

const retryLimit = 3

func (rc retryCounter) maybe(err error, reqCh chan Request) bool {
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
		defer catchAll("retrying: " + rt.req.URL)
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
	if err := recover(); err != nil {
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
	/*
		for _, f := range fdDir {
			n, e := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, f.Name()))
			if e != nil {
				n = f.Name()
			}

			Log.Vomit.Printf("%s: %s", f.Mode(), n)
		}
	*/

	Log.Debug.Printf("%s: %d", when, len(fdDir))
}

func singPipeline(
	url string,
	client *singularity.Client,
	dw, wg *sync.WaitGroup,
	reqs chan Request,
	errs chan error,
	clusters []string,
) {
	Log.Vomit.Printf("Starting cluster at %s", url)
	defer func() { Log.Vomit.Printf("Completed cluster at %s", url) }()
	defer wg.Done()
	defer catchAndSend(fmt.Sprintf("get requests: %s", url), errs)
	rs, err := getRequestsFromSingularity(url, client, clusters)
	if err != nil {
		Log.Vomit.Print(err) //XXX connection reset by peer should be retried
		errs <- errors.Wrap(err, "getting request list")
		return
	}
	for _, r := range rs {
		Log.Vomit.Printf("Req: %s %s %d", r.URL, reqID(r.RequestParent), r.RequestParent.Request.Instances)
		dw.Add(1)
		reqs <- r
	}
}

func getRequestsFromSingularity(url string, client *singularity.Client, clusters []string) ([]Request, error) {
	logFDs("before getRequestsFromSingularity")
	defer logFDs("after getRequestsFromSingularity")
	singRequests, err := client.GetRequests()
	if err != nil {
		return nil, errors.Wrap(err, "getting request")
	}

	reqs := make([]Request, 0, len(singRequests))
eachrequest:
	for _, sr := range singRequests {
		// Parse requests, filter out malformed ones and those that do not
		// belong to one of the specified clusters.
		deployID, err := ParseRequestID(sr.Request.Id)
		if err != nil {
			Log.Vomit.Printf("Ignoring Singularity Request %q: %s", sr.Request.Id, err)
			continue
		}
		// Pick out the request which matches this cluster.
		for _, c := range clusters {
			if deployID.Cluster == c {
				request := buildRequest(url, client, sr)
				reqs = append(reqs, request)
				continue eachrequest
			}
		}
		Log.Debug.Printf("ignoring request %q as it's not one of my clusters (%# v)", sr.Request.Id, clusters)
	}

	return reqs, nil
}

func buildRequest(url string, client *singularity.Client, rp *dtos.SingularityRequestParent) Request {
	return Request{
		URL:           url,
		Client:        client,
		RequestParent: rp,
	}
}

func depPipeline(
	cl rectificationClient,
	reg sous.Registry,
	clusters sous.Clusters,
	poolCount int,
	reqCh chan Request,
	depCh chan *sous.DeployState,
	errCh chan error,
) {
	defer catchAndSend("dependency building", errCh)
	poolLimit := make(chan struct{}, poolCount)
	for req := range reqCh {
		Log.Vomit.Printf("starting assembling for %q", reqID(req.RequestParent))
		go func(req Request) {
			defer catchAndSend(fmt.Sprintf("dep from req %s", req.URL), errCh)

			poolLimit <- struct{}{}
			defer func() {
				Log.Vomit.Printf("finished assembling for %q", reqID(req.RequestParent))
				<-poolLimit
			}()

			dep, err := assembleDeployment(cl, reg, clusters, req)

			if err != nil {
				errCh <- errors.Wrap(err, "assembly problem")
			} else {
				depCh <- dep
			}
		}(req)
	}
}

func assembleDeployment(cl rectificationClient, reg sous.Registry, clusters sous.Clusters, req Request) (*sous.DeployState, error) {
	Log.Vomit.Printf("Assembling from: %s %s", req.URL, reqID(req.RequestParent))
	tgt, err := BuildDeployment(reg, clusters, req)

	if err != nil {
		return nil, errors.Wrap(err, "Building deployment")
	}

	Log.Vomit.Printf("Collected deployment: %v", tgt)
	return &tgt, nil
}
