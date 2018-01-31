package server

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestGETDeploymentHandler_Exchange(t *testing.T) {

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
		gdh := &GETDeploymentHandler{
			QueueSet:     queues,
			DeploymentID: newDid("nonexistent"),
		}
		body, gotStatus := gdh.Exchange()
		gotResponse := body.(deploymentResponse)
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
	t.Run("one_queued", func(t *testing.T) {
		gdh := &GETDeploymentHandler{
			QueueSet:     queues,
			DeploymentID: newDid("one"),
		}
		body, gotStatus := gdh.Exchange()
		gotResponse := body.(deploymentResponse)
		const wantStatus = 200
		if gotStatus != wantStatus {
			t.Errorf("got status %d; want %d", gotStatus, wantStatus)
		}
		gotLen := len(gotResponse.Queue)
		wantLen := 1
		if gotLen != wantLen {
			t.Errorf("got %d queued; want %d", gotLen, wantLen)
		}
		item := gotResponse.Queue[0]
		wantR11nID := queuedOne1.ID
		gotR11nID := item.ID
		if gotR11nID != wantR11nID {
			t.Errorf("got R11nID %q; want %q", gotR11nID, wantR11nID)
		}

	})
	t.Run("two_queued", func(t *testing.T) {
		gdh := &GETDeploymentHandler{
			QueueSet:     queues,
			DeploymentID: newDid("two"),
		}
		body, gotStatus := gdh.Exchange()
		gotResponse := body.(deploymentResponse)
		const wantStatus = 200
		if gotStatus != wantStatus {
			t.Errorf("got status %d; want %d", gotStatus, wantStatus)
		}
		gotLen := len(gotResponse.Queue)
		wantLen := 2
		if gotLen != wantLen {
			t.Errorf("got %d queued; want %d", gotLen, wantLen)
		}

		{
			item := gotResponse.Queue[0]
			wantR11nID := queuedTwo1.ID
			gotR11nID := item.ID
			if gotR11nID != wantR11nID {
				t.Errorf("got R11nID %q; want %q", gotR11nID, wantR11nID)
			}
		}
		{
			item := gotResponse.Queue[1]
			wantR11nID := queuedTwo2.ID
			gotR11nID := item.ID
			if gotR11nID != wantR11nID {
				t.Errorf("got R11nID %q; want %q", gotR11nID, wantR11nID)
			}
		}
	})

}

func newR11n(repo string) *sous.Rectification {
	r11n := &sous.Rectification{
		Pair: sous.DeployablePair{},
	}
	r11n.Pair.SetID(newDid(repo))
	return r11n
}
