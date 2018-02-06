package server

import (
	"fmt"
	"testing"

	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
)

func TestR11nResource_Get_no_errors(t *testing.T) {

	testCases := []struct {
		desc       string
		params     httprouter.Params
		wantDID    sous.DeploymentID
		wantR11nID sous.R11nID
	}{
		{
			desc: "valid deploymentID and r11nID",
			params: httprouter.Params{
				{Key: "DeploymentID", Value: "cluster1%3Agithub.com%2Fuser1%2Frepo1%2Cdir1~flavor1"},
				{Key: "R11nID", Value: "cabba9e"},
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
			wantR11nID: sous.R11nID("cabba9e"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dr := &R11nResource{}
			got := dr.Get(nil, nil, tc.params).(*GETR11nHandler)

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
	gdh := &GETDeploymentHandler{
		DeploymentIDErr: fmt.Errorf("this error"),
	}
	_, gotStatus := gdh.Exchange()
	const wantStatus = 404
	if gotStatus != wantStatus {
		t.Errorf("got status %d; want %d", gotStatus, wantStatus)
	}
}
