package server

import (
	"fmt"
	"testing"

	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
)

func TestDeploymentResource_Get_no_errors(t *testing.T) {

	testCases := []struct {
		desc    string
		params  httprouter.Params
		wantDID sous.DeploymentID
	}{
		{
			desc: "valid deploymentID",
			params: httprouter.Params{
				{Key: "DeploymentID", Value: "cluster1%3Agithub.com%2Fuser1%2Frepo1%2Cdir1~flavor1"},
			},
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
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dr := &DeploymentResource{}
			got := dr.Get(nil, nil, tc.params).(*GETDeploymentHandler)

			gotDID := got.DeploymentID
			if gotDID != tc.wantDID {
				t.Errorf("got DeploymentID:\n%#v; want:\n%#v", gotDID, tc.wantDID)
			}

			if got.DeploymentIDErr != nil {
				t.Errorf("unexpected error: %s", got.DeploymentIDErr)
			}
		})
	}

}

func TestDeploymentResource_Get_DeploymentID_errors(t *testing.T) {

	testCases := []struct {
		params     httprouter.Params
		wantDIDErr string
	}{
		{
			params: httprouter.Params{
				{Key: "DeploymentID", Value: "cluster1%-3Agithub.com%2Fuser1%2Frepo1%2Cdir1~flavor1"},
			},
			wantDIDErr: `invalid URL escape "%-3"`,
		},
		{
			params: httprouter.Params{
				{Key: "DeploymentID", Value: "cluster1Agithub.com%2Fuser1%2Frepo1%2Cdir1~flavor1"},
			},
			wantDIDErr: `does not contain a colon`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.wantDIDErr, func(t *testing.T) {
			dr := &DeploymentResource{}
			got := dr.Get(nil, nil, tc.params).(*GETDeploymentHandler)

			gotDIDErr := got.DeploymentIDErr
			if gotDIDErr == nil || gotDIDErr.Error() != tc.wantDIDErr {
				t.Fatalf("got error: %v; want %q", gotDIDErr, tc.wantDIDErr)
			}
		})
	}

}

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

func TestGETDeploymentHandler_Exchange_errors(t *testing.T) {
	gdh := &GETDeploymentHandler{
		DeploymentIDErr: fmt.Errorf("this error"),
	}
	_, gotStatus := gdh.Exchange()
	const wantStatus = 404
	if gotStatus != wantStatus {
		t.Errorf("got status %d; want %d", gotStatus, wantStatus)
	}
}

func newR11n(repo string) *sous.Rectification {
	r11n := &sous.Rectification{
		Pair: sous.DeployablePair{},
	}
	r11n.Pair.SetID(newDid(repo))
	return r11n
}
