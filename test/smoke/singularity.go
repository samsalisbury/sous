package smoke

import (
	"fmt"
	"log"
	"os"
	"sort"
	"testing"
	"time"

	sing "github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/singularity"
	sous "github.com/opentable/sous/lib"
)

type testSingularity struct {
	URL           string
	client        *sing.Client
	ClusterSuffix string
}

func newSingularity(baseURL string) *testSingularity {
	return &testSingularity{URL: baseURL, client: sing.NewClient(baseURL)}
}

func (s *testSingularity) PauseRequestForDeployment(t *testing.T, reqID string) {
	t.Helper()
	if _, err := s.client.Pause(reqID, nil); err != nil {
		t.Fatal(err)
	}

	var depID string
	waitFor(t, "paused status", 30*time.Second, 2*time.Second, func() error {
		req, err := s.client.GetRequest(reqID, false)
		if req.ActiveDeploy != nil && req.ActiveDeploy.Id != "" {
			depID = req.ActiveDeploy.Id
		}
		if err != nil {
			return err
		}
		if req.State != dtos.SingularityRequestParentRequestStatePAUSED {
			return fmt.Errorf("status is %s", req.State)
		}
		return nil
	})

	waitFor(t, "tasks to stop", 30*time.Second, 2*time.Second, func() error {
		h, err := s.client.GetActiveDeployTasks(reqID, depID)
		if err != nil {
			return err
		}
		if len(h) != 0 {
			return fmt.Errorf("%d tasks running", len(h))
		}
		return nil
	})
}

func (s *testSingularity) UnpauseRequestForDeployment(t *testing.T, reqID string) {
	t.Helper()
	if _, err := s.client.Unpause(reqID, nil); err != nil {
		t.Fatal(err)
	}

	waitFor(t, "not paused status", 30*time.Second, 2*time.Second, func() error {
		req, err := s.client.GetRequest(reqID, false)
		if err != nil {
			return err
		}
		if req.State == dtos.SingularityRequestParentRequestStatePAUSED {
			return fmt.Errorf("status is %s", req.State)
		}
		return nil
	})
}

func (s *testSingularity) MustGetRequestForDeployment(t *testing.T, reqID string) *dtos.SingularityRequestParent {
	t.Helper()
	req, err := s.GetRequestForDeployment(t, reqID)
	if err != nil {
		t.Fatalf("getting request: %s", err)
	}
	return req
}

func (s *testSingularity) GetRequestForDeployment(t *testing.T, reqID string) (*dtos.SingularityRequestParent, error) {
	t.Helper()
	return s.client.GetRequest(reqID, false)
}

func (s *testSingularity) MustGetLatestDeployForDeployment(t *testing.T, reqID string) *dtos.SingularityDeployHistory {
	t.Helper()
	deps, err := s.client.GetDeploys(reqID, 100, 1)
	if err != nil {
		t.Fatalf("getting deployments for request: %s", err)
	}
	if len(deps) == 0 {
		t.Fatalf("zero deployments for request %q", reqID)
	}

	// Sort by timestamp descending.
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].DeployMarker.Timestamp > deps[j].DeployMarker.Timestamp
	})

	// Fetch the actual deploy (this one has a nil Deploy field.
	deployID := deps[0].DeployMarker.DeployId
	dep, err := s.client.GetDeploy(reqID, deployID)
	if err != nil {
		t.Fatalf("getting deploy detail for %q", deployID)
	}

	return dep
}

func (s *testSingularity) DefaultReqID(t *testing.T, did sous.DeploymentID) string {
	t.Helper()
	did.Cluster = did.Cluster + s.ClusterSuffix
	reqID, err := singularity.MakeRequestID(did)
	if err != nil {
		t.Fatalf("making singularity request ID: %s", err)
	}
	return reqID
}

func waitFor(t *testing.T, what string, timeout, interval time.Duration, f func() error) {
	t.Helper()
	fmt.Fprintf(os.Stderr, "waitFor: Waiting for %s...\n", what)
	ticker := time.NewTicker(interval)
	startTime := time.Now()
	defer ticker.Stop()
	select {
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for %s after %s", what, timeout)
	case <-(func() <-chan struct{} {
		c := make(chan struct{})
		go func() {
			defer close(c)
			for {
				err := func() error {
					select {
					case <-ticker.C:
						return fmt.Errorf("timed out after %s", interval)
					case err := <-(func() <-chan error {
						c := make(chan error)
						go func() { c <- f() }()
						return c
					}()):
						return err
					}
				}()
				if err != nil {
					// Log direct to stderr for live updates.
					elapsed := time.Now().Sub(startTime).Round(time.Second)
					fmt.Fprintf(os.Stderr, "waitFor: Waiting for %s: %s (%s elapsed)\n", what, err, elapsed)
					<-ticker.C
					continue
				}
				break
			}
		}()
		return c
	}()):
	}
}

func (s *testSingularity) Reset() error {
	const pollLimit = 30
	const retryLimit = 3
	log.Printf("Resetting Singularity...")

	reqList, err := s.client.GetRequests(false)
	if err != nil {
		panic(err)
	}

	// Singularity is sometimes not actually deleting a request until the second attempt...
	for j := retryLimit; j >= 0; j-- {
		for _, r := range reqList {
			_, err := s.client.DeleteRequest(r.Request.Id, nil)
			if err != nil {
				panic(err)
			}
		}

		log.Printf("Singularity resetting: Issued deletes for %d requests. Awaiting confirmation they've quit.", len(reqList))

		for i := pollLimit; i > 0; i-- {
			reqList, err = s.client.GetRequests(false)
			if err != nil {
				panic(err)
			}
			if len(reqList) == 0 {
				log.Printf("Singularity successfully reset.")
				return nil
			}
			time.Sleep(time.Second)
		}
	}
	for n, req := range reqList {
		log.Printf("Singularity reset failure: stubborn request: #%d/%d %#v", n+1, len(reqList), req)
	}
	return fmt.Errorf("singularity not reset after %d * %d tries - %d requests remain", retryLimit, pollLimit, len(reqList))
}
