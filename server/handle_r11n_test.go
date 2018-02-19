package server

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	sous "github.com/opentable/sous/lib"
)

// TestNewR11nResource checks that the same queue set passed to the
// constructor makes its way to the get handler.
func TestNewR11nResource(t *testing.T) {
	qs := &sous.R11nQueueSet{}
	c := ComponentLocator{
		QueueSet: qs,
	}
	dq := newR11nResource(c)

	got := dq.Get(nil, &http.Request{URL: &url.URL{}}, nil).(*GETR11nHandler)
	if got.QueueSet != qs {
		t.Errorf("got different queueset")
	}
}

func TestR11nResource_Get_no_errors(t *testing.T) {

	testCases := []struct {
		desc       string
		query      string
		wantDID    sous.DeploymentID
		wantR11nID sous.R11nID
		wantWait   bool
	}{
		{
			desc:  "valid deploymentID and r11nID",
			query: "cluster=cluster1&repo=github.com%2Fuser1%2Frepo1&offset=dir1&flavor=flavor1&R11nID=cabba9e",
			wantDID: sous.DeploymentID{
				ManifestID: sous.ManifestID{
					Source: sous.SourceLocation{
						Repo: "github.com/user1/repo1",
						Dir:  "dir1",
					},
					Flavor: "flavor1",
				},
				Cluster: "cluster1",
			},
			wantR11nID: sous.R11nID("cabba9e"),
		},
		{
			desc:  "valid short DeploymentID and r11nID",
			query: "cluster=cluster1&repo=github.com%2Fuser1%2Frepo1&R11nID=cabba9e",
			wantDID: sous.DeploymentID{
				ManifestID: sous.ManifestID{
					Source: sous.SourceLocation{
						Repo: "github.com/user1/repo1",
					},
				},
				Cluster: "cluster1",
			},
			wantR11nID: sous.R11nID("cabba9e"),
		},
		{
			desc:  "wait query",
			query: "cluster=cluster1&repo=github.com%2Fuser1%2Frepo1&R11nID=cabba9e&wait=true",
			wantDID: sous.DeploymentID{
				ManifestID: sous.ManifestID{
					Source: sous.SourceLocation{
						Repo: "github.com/user1/repo1",
					},
				},
				Cluster: "cluster1",
			},
			wantR11nID: sous.R11nID("cabba9e"),
			wantWait:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dr := &R11nResource{}
			req := makeRequestWithQuery(t, tc.query)

			got := dr.Get(nil, req, nil).(*GETR11nHandler)

			gotDID := got.DeploymentID
			if gotDID != tc.wantDID {
				t.Errorf("got DeploymentID:\n%#v; want:\n%#v", gotDID, tc.wantDID)
			}
			if got.DeploymentIDErr != nil {
				t.Errorf("unexpected error: %s", got.DeploymentIDErr)
			}

			gotRID := got.R11nID
			if gotRID != tc.wantR11nID {
				t.Errorf("got R11nID %q; want %q", gotRID, tc.wantR11nID)
			}

			gotWait := got.WaitForResolution
			wantWait := tc.wantWait
			if gotWait != wantWait {
				t.Errorf("got WaitForResolution %t; want %t", gotWait, wantWait)
			}
		})
	}

}

func TestGETR11nHandler_Exchange(t *testing.T) {

	queues := sous.NewR11nQueueSet()
	queuedOne1, ok := queues.Push(newR11n("one"))
	if !ok {
		t.Fatal("setup failed to push r11n")
	}
	queuedTwo1, ok := queues.Push(newR11n("two"))
	if !ok {
		t.Fatal("setup failed to push r11n")
	}
	queuedTwo2, ok := queues.Push(newR11n("two"))
	if !ok {
		t.Fatal("setup failed to push r11n")
	}

	t.Run("nonexistent_deployID", func(t *testing.T) {
		gdh := &GETR11nHandler{
			QueueSet:     queues,
			DeploymentID: newDid("nonexistent"),
		}
		body, gotStatus := gdh.Exchange()
		gotResponse := body.(deployQueueResponse)
		const wantStatus = 404
		if gotStatus != wantStatus {
			t.Errorf("got status %d; want %d", gotStatus, wantStatus)
		}
		gotLen := len(gotResponse.Queue)
		wantLen := 0
		if gotLen != wantLen {
			t.Errorf("got %d queued; want %d", gotLen, wantLen)
		}
	})
	t.Run("nonexistent_r11nID", func(t *testing.T) {
		gdh := &GETR11nHandler{
			QueueSet:     queues,
			DeploymentID: newDid("one"),
			R11nID:       "nonexistent",
		}
		_, gotStatus := gdh.Exchange()
		const wantStatus = 404
		if gotStatus != wantStatus {
			t.Errorf("got status %d; want %d", gotStatus, wantStatus)
		}
	})
	t.Run("one_1", func(t *testing.T) {
		gdh := &GETR11nHandler{
			QueueSet:     queues,
			DeploymentID: newDid("one"),
			R11nID:       queuedOne1.ID,
		}
		body, gotStatus := gdh.Exchange()
		gotResponse := body.(r11nResponse)
		const wantStatus = 200
		if gotStatus != wantStatus {
			t.Errorf("got status %d; want %d", gotStatus, wantStatus)
		}
		gotPos := gotResponse.QueuePosition
		const wantPos = 0
		if gotPos != wantPos {
			t.Errorf("got queue position %d; want %d", gotPos, wantPos)

		}
	})
	t.Run("two_1", func(t *testing.T) {
		gdh := &GETR11nHandler{
			QueueSet:     queues,
			DeploymentID: newDid("two"),
			R11nID:       queuedTwo1.ID,
		}
		body, gotStatus := gdh.Exchange()
		gotResponse := body.(r11nResponse)
		const wantStatus = 200
		if gotStatus != wantStatus {
			t.Errorf("got status %d; want %d", gotStatus, wantStatus)
		}
		gotPos := gotResponse.QueuePosition
		const wantPos = 0
		if gotPos != wantPos {
			t.Errorf("got queue position %d; want %d", gotPos, wantPos)
		}
	})
	t.Run("two_2", func(t *testing.T) {
		gdh := &GETR11nHandler{
			QueueSet:     queues,
			DeploymentID: newDid("two"),
			R11nID:       queuedTwo2.ID,
		}
		body, gotStatus := gdh.Exchange()
		gotResponse := body.(r11nResponse)
		const wantStatus = 200
		if gotStatus != wantStatus {
			t.Errorf("got status %d; want %d", gotStatus, wantStatus)
		}
		gotPos := gotResponse.QueuePosition
		const wantPos = 1
		if gotPos != wantPos {
			t.Errorf("got queue position %d; want %d", gotPos, wantPos)

		}
	})
}

func TestGETR11nHandler_Exchange_errors(t *testing.T) {
	gdh := &GETR11nHandler{
		DeploymentIDErr: fmt.Errorf("this error"),
	}
	_, gotStatus := gdh.Exchange()
	const wantStatus = 404
	if gotStatus != wantStatus {
		t.Errorf("got status %d; want %d", gotStatus, wantStatus)
	}
}

// TestGETR11nHandler_Exchange_wait_success checks that when WaitForResolution
// is true, Exchange does not return until the queue reports that the r11n is
// done, and that it returns the expected Resolution result.
func TestGETR11nHandler_Exchange_wait_success(t *testing.T) {
	block := make(chan struct{})
	qh := func(qr *sous.QueuedR11n) sous.DiffResolution {
		<-block
		return sous.DiffResolution{
			DeploymentID: newDid("one"),
			Desc:         "desc1",
		}
	}
	queues := sous.NewR11nQueueSet(sous.R11nQueueStartWithHandler(qh))
	queuedOne1, ok := queues.Push(newR11n("one"))
	if !ok {
		t.Fatal("setup failed to push r11n")
	}

	grh := &GETR11nHandler{
		WaitForResolution: true,
		QueueSet:          queues,
		DeploymentID:      newDid("one"),
		R11nID:            queuedOne1.ID,
	}

	// response wraps up the pair of return parameters from Exchange.
	type response struct {
		status int
		body   interface{}
	}
	// Once Exchange completes, write its result to responses.
	responses := make(chan response)
	go func() {
		r, s := grh.Exchange()
		responses <- response{status: s, body: r}
	}()

	// At this point responses should not emit anything because block will be
	// blocking the queue from being processed, which should block Exchange from
	// returning due to grh.WaitForResolution == true.
	timeout := 10 * time.Millisecond
	select {
	// We may sometimes get away with default instead of this timeout here, but
	// the short timeout gives Exchange a little more time to complete, and
	// should result in fewer false passes for this assertion.
	case <-time.After(timeout): // OK
	case <-responses:
		t.Fatalf("Exchange returned before %s, should have waited for resolution.", timeout)
	}

	// Close block to allow the queue processor to proceed.
	close(block)
	select {
	case <-time.After(timeout):
		t.Fatalf("Exchange did not return within %s of being unblocked.", timeout)
	case got := <-responses: // OK
		const wantStatus = 200
		if got.status != wantStatus {
			t.Errorf("got status %d; want %d", got.status, wantStatus)
		}
		gotBody := got.body.(r11nResponse)
		if gotBody.Resolution == nil {
			t.Fatalf("unexpected nil Resolution")
		}
		gotDID := gotBody.Resolution.DeploymentID
		wantDID := newDid("one")
		if gotDID != wantDID {
			t.Errorf("got DeploymentID %#v; want %#v", gotDID, wantDID)
		}
		gotDesc := gotBody.Resolution.Desc
		wantDesc := sous.ResolutionType("desc1")
		if gotDesc != wantDesc {
			t.Errorf("got desc %q; want %q", gotDesc, wantDesc)
		}
	}
}
