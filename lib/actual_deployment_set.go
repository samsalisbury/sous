package sous

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
)

// ReqsPerServer limits the number of simultaneous number of requests made
// against a single Singularity server
const ReqsPerServer = 10

type (
	// SetCollector is the agent responsible for collecting sets of deployments
	// For now, it can collect the actual running set
	SetCollector struct {
		rectClient RectificationClient
	}

	sDeploy    *dtos.SingularityDeploy
	sRequest   *dtos.SingularityRequest
	sDepMarker *dtos.SingularityDeployMarker

	SingReq struct {
		SourceURL string
		Sing      *singularity.Client
		ReqParent *dtos.SingularityRequestParent
	}

	retryCounter map[string]uint
)

// NewSetCollector returns a new set collector
func NewSetCollector(rc RectificationClient) *SetCollector {
	return &SetCollector{rc}
}

// GetRunningDeployment collects data from the Singularity clusters and
// returns a list of actual deployments
func (sc *SetCollector) GetRunningDeployment(singUrls []string) (deps Deployments, err error) {
	retries := make(retryCounter)
	errCh := make(chan error)
	deps = make(Deployments, 0)
	sings := make(map[string]*singularity.Client)
	reqCh := make(chan SingReq, len(singUrls)*ReqsPerServer)
	depCh := make(chan *Deployment, ReqsPerServer)

	defer close(depCh)
	// XXX The intention here was to use something like the gotools context to
	// manage NW cancellation
	//defer sc.rectClient.Cancel()

	var singWait, depWait sync.WaitGroup

	singWait.Add(len(singUrls))
	for _, url := range singUrls {
		sing := singularity.NewClient(url)
		sings[url] = sing
		go singPipeline(sing, &depWait, &singWait, reqCh, errCh)
	}

	go depPipeline(sc.rectClient, reqCh, depCh, errCh)

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
			Log.Debug.Print(dep)
			deps = append(deps, dep)
			depWait.Done()
		case err = <-errCh:
			if _, ok := err.(malformedResponse); ok {
				Log.Info.Print(err)
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
		errs <- err
		return
	}
	for _, r := range rs {
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
	cl RectificationClient,
	reqCh chan SingReq,
	depCh chan *Deployment,
	errCh chan error,
) {
	defer catchAndSend("dependency building", errCh)
	for req := range reqCh {
		go func(cl RectificationClient, req SingReq) {
			defer catchAndSend(fmt.Sprintf("dep from req %s", req.SourceURL), errCh)

			dep, err := assembleDeployment(cl, req)

			if err != nil {
				errCh <- err
			} else {
				Log.Debug.Print(dep)
				depCh <- dep
			}
		}(cl, req)
	}
}

func assembleDeployment(cl RectificationClient, req SingReq) (*Deployment, error) {
	uc := NewDeploymentBuilder(cl, req)
	err := uc.CompleteConstruction()
	if err != nil {
		return nil, err
	}

	Log.Debug.Print(uc)
	return &uc.Target, nil
}
