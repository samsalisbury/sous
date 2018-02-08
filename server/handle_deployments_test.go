package server

import (
	"testing"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/pborman/uuid"
)

func TestGETDeploymentsHandler_Exchange(t *testing.T) {

	testCases := []struct {
		desc       string
		dids       []sous.DeploymentID
		wantResult deploymentsResponse
	}{
		{
			desc:       "empty queue",
			dids:       []sous.DeploymentID{},
			wantResult: deploymentsResponse{Deployments: map[sous.DeploymentID]int{}},
		},
		{
			desc: "one DeploymentID",
			dids: []sous.DeploymentID{newDid("one")},
			wantResult: deploymentsResponse{Deployments: map[sous.DeploymentID]int{
				newDid("one"): 1,
			}},
		},
		{
			desc: "two unique DeploymentIDs",
			dids: []sous.DeploymentID{newDid("one"), newDid("two")},
			wantResult: deploymentsResponse{Deployments: map[sous.DeploymentID]int{
				newDid("one"): 1,
				newDid("two"): 1,
			}},
		},
		{
			desc: "same DeploymentID twice",
			dids: []sous.DeploymentID{newDid("one"), newDid("one")},
			wantResult: deploymentsResponse{Deployments: map[sous.DeploymentID]int{
				newDid("one"): 2,
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			qs := sous.NewR11nQueueSet()

			for _, did := range tc.dids {
				r11n := &sous.Rectification{
					Pair: sous.DeployablePair{},
				}

				r11n.Pair.SetID(did)

				if _, ok := qs.Push(r11n); !ok {
					t.Fatal("precondition failed: failed to push r11n")
				}

			}

			handler := &GETDeploymentsHandler{
				QueueSet: qs,
			}

			data, gotStatusCode := handler.Exchange()

			const wantStatusCode = 200
			if gotStatusCode != wantStatusCode {
				t.Errorf("got %d; want %d", gotStatusCode, wantStatusCode)
			}

			dr, ok := data.(deploymentsResponse)
			if !ok {
				t.Fatalf("got a %T; want a %T", data, dr)
			}

			wantLen := len(tc.wantResult.Deployments)
			gotLen := len(dr.Deployments)
			if gotLen != wantLen {
				t.Fatalf("got %d queued deployments; want %d", gotLen, wantLen)
			}

			for did, wantCount := range tc.wantResult.Deployments {
				gotCount := dr.Deployments[did]
				if gotCount != wantCount {
					t.Errorf("got %d queued rectifications for %q; want %d", gotCount, did, wantCount)
				}
			}

			testCount := dr.Deployments[sous.DeploymentID{}]
			if testCount != 0 {
				t.Errorf("got %d for empty DeploymentID expected 0", testCount)
			}
		})
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
func TestGETDeploymentsHandler_Exchange_async(t *testing.T) {

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
	dh := GETDeploymentsHandler{QueueSet: queues}

	// Start calling Exchange in a hot loop.
	for i := 0; i < 1000; i++ {
		dh.Exchange()
	}
}
