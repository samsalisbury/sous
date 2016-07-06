package singularity

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
)

// ReqsPerServer limits the number of simultaneous number of requests made
// against a single Singularity server
const ReqsPerServer = 10

type (
	// SetCollector implements sous.Deployer
	SetCollector struct {
		sous.RectificationClient
	}

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

// NewSetCollector returns a new set collector
func NewSetCollector(rc sous.RectificationClient) *SetCollector {
	return &SetCollector{rc}
}

// GetRunningDeployment collects data from the Singularity clusters and
// returns a list of actual deployments
func (sc *SetCollector) GetRunningDeployment(singUrls []string) (deps sous.Deployments, err error) {
	retries := make(retryCounter)
	errCh := make(chan error)
	deps = make(sous.Deployments, 0)
	sings := make(map[string]*singularity.Client)
	reqCh := make(chan SingReq, len(singUrls)*ReqsPerServer)
	depCh := make(chan *sous.Deployment, ReqsPerServer)

	defer close(depCh)
	// XXX The intention here was to use something like the gotools context to
	// manage NW cancellation
	//defer sc.rectClient.Cancel()

	var singWait, depWait sync.WaitGroup

	singWait.Add(len(singUrls))
	for _, url := range singUrls {
		sing := singularity.NewClient(url)
		//sing.Debug = true
		sings[url] = sing
		go singPipeline(sing, &depWait, &singWait, reqCh, errCh)
	}

	go depPipeline(sc.RectificationClient, reqCh, depCh, errCh)

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

func catchAndSend(from string, errs chan error) {
	defer catchAll(from)
	if err := recover(); err != nil {
		Log.Debug.Printf("from = %s err = %+v\n", from, err)
		Log.Debug.Printf("debug.Stack() = %+v\n", string(debug.Stack()))
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

func singPipeline(
	client *singularity.Client,
	dw, wg *sync.WaitGroup,
	reqs chan SingReq,
	errs chan error,
) {
	defer wg.Done()
	defer catchAndSend(fmt.Sprintf("get requests: %s", client), errs)
	rs, err := getRequestsFromSingularity(client)
	if err != nil {
		Log.Vomit.Print(err)
		errs <- err
		return
	}
	for _, r := range rs {
		Log.Vomit.Print("Req: ", r)
		dw.Add(1)
		reqs <- r
	}
}

func getRequestsFromSingularity(client *singularity.Client) ([]SingReq, error) {
	singRequests, err := client.GetRequests()
	if err != nil {
		return nil, err
	}

	reqs := make([]SingReq, 0, len(singRequests))
	for _, sr := range singRequests {
		reqs = append(reqs, SingReq{client.BaseUrl, client, sr})
	}

	return reqs, nil
}

func depPipeline(
	cl sous.RectificationClient,
	reqCh chan SingReq,
	depCh chan *sous.Deployment,
	errCh chan error,
) {
	defer catchAndSend("dependency building", errCh)
	for req := range reqCh {
		go func(cl sous.RectificationClient, req SingReq) {
			defer catchAndSend(fmt.Sprintf("dep from req %s", req.SourceURL), errCh)

			dep, err := assembleDeployment(cl, req)

			if err != nil {
				errCh <- err
			} else {
				depCh <- dep
			}
		}(cl, req)
	}
}

func assembleDeployment(cl sous.RectificationClient, req SingReq) (*sous.Deployment, error) {
	Log.Vomit.Print("Assembling from: ", req)
	uc := NewDeploymentBuilder(cl, req)
	err := uc.CompleteConstruction()
	if err != nil {
		Log.Vomit.Print(err)
		return nil, err
	}

	Log.Vomit.Printf("Collected deployment: %v", uc)
	return &uc.Target, nil
}
