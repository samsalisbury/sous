package server

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/pborman/uuid"
)

// TestNewAllDeployQueuesResource checks that the same queue set passed to the
// constructor makes its way to the get handler.
func TestNewAllDeployQueuesResource(t *testing.T) {
	qs := &sous.R11nQueueSet{}
	c := ComponentLocator{
		QueueSet: qs,
	}
	adq := newAllDeployQueuesResource(c)

	got := adq.Get(nil, &http.Request{URL: &url.URL{}}, nil).(*GETAllDeployQueuesHandler)
	if got.QueueSet != qs {
		t.Errorf("got different queueset")
	}
}

func TestGETAllDeployQueuesHandler_Exchange(t *testing.T) {
	t.Run("empty queue", func(t *testing.T) {
		data, status := setupExchange(t)
		assertSuccess(t, status)
		dqr := assertIsDeploymentQueuesResponse(t, data)
		assertQueueLength(t, dqr, sous.DeploymentID{}, 0)
		assertNumQueues(t, dqr, 0)
	})

	t.Run("one DeploymentID", func(t *testing.T) {
		data, status := setupExchange(t, newDid("one"))
		assertSuccess(t, status)
		dqr := assertIsDeploymentQueuesResponse(t, data)
		assertQueueLength(t, dqr, sous.DeploymentID{}, 0)
		assertNumQueues(t, dqr, 1)
		assertQueueLength(t, dqr, newDid("one"), 1)
	})

	t.Run("two unique DeploymentIDs", func(t *testing.T) {
		data, status := setupExchange(t, newDid("one"), newDid("two"))
		assertSuccess(t, status)
		dqr := assertIsDeploymentQueuesResponse(t, data)
		assertQueueLength(t, dqr, sous.DeploymentID{}, 0)
		assertNumQueues(t, dqr, 2)
		assertQueueLength(t, dqr, newDid("one"), 1)
		assertQueueLength(t, dqr, newDid("two"), 1)
	})

	t.Run("same deployment twice", func(t *testing.T) {
		data, status := setupExchange(t, newDid("one"), newDid("one"))
		assertSuccess(t, status)
		dqr := assertIsDeploymentQueuesResponse(t, data)
		assertQueueLength(t, dqr, sous.DeploymentID{}, 0)
		assertNumQueues(t, dqr, 1)
		assertQueueLength(t, dqr, newDid("one"), 2)
	})

}

func setupExchange(t *testing.T, dids ...sous.DeploymentID) (interface{}, int) {
	qs := sous.NewR11nQueueSet()

	for _, did := range dids {
		r11n := &sous.Rectification{
			Pair: sous.DeployablePair{},
		}

		r11n.Pair.SetID(did)

		if _, ok := qs.Push(r11n); !ok {
			t.Fatal("precondition failed: failed to push r11n")
		}

	}

	handler := &GETAllDeployQueuesHandler{
		QueueSet: qs,
	}

	return handler.Exchange()
}

func assertSuccess(t *testing.T, status int) {
	const wantStatusCode = 200
	if status != wantStatusCode {
		t.Errorf("got %d; want %d", status, wantStatusCode)
	}

}

func assertIsDeploymentQueuesResponse(t *testing.T, data interface{}) DeploymentQueuesResponse {
	dr, ok := data.(DeploymentQueuesResponse)
	if !ok {
		t.Fatalf("got a %T; want a %T", data, dr)
		return DeploymentQueuesResponse{}
	}
	return dr
}

func assertNumQueues(t *testing.T, dr DeploymentQueuesResponse, wantLen int) {
	gotLen := len(dr.Queues)
	if gotLen != wantLen {
		t.Fatalf("got %d queued deployments; want %d", gotLen, wantLen)
	}
}

func assertQueueLength(t *testing.T, dr DeploymentQueuesResponse, did sous.DeploymentID, wantCount int) {
	gotCount := dr.Queues[did.String()].Length
	if gotCount != wantCount {
		t.Errorf("got %d queued rectifications for %q; want %d", gotCount, did.String(), wantCount)
	}
}

func newDid(repo string) sous.DeploymentID {
	return sous.DeploymentID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: repo,
			},
		},
	}
}

// TestGETDeploymentsHandler_Exchange_async should be run with the -race flag.
func TestGETAllDeployQueuesHandler_Exchange_async(t *testing.T) {

	// Start writing to a new queueset that's also being processed in a hot loop.
	qh := func(*sous.QueuedR11n) sous.DiffResolution { return sous.DiffResolution{} }
	queues := sous.NewR11nQueueSet(sous.R11nQueueStartWithHandler(qh))
	go func() {
		for {
			did := newDid(uuid.New())
			did.Cluster = uuid.New()
			r11n := newR11n("")
			r11n.Pair.SetID(did)
			queues.Push(r11n)
			time.Sleep(time.Millisecond)
		}
	}()

	// Set up a handler using the above busy queue set.
	dh := GETAllDeployQueuesHandler{QueueSet: queues}

	// Start calling Exchange in a hot loop.
	for i := 0; i < 1000; i++ {
		dh.Exchange()
	}
}
