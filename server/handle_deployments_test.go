package server

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestHandleDeployments_Exchange(t *testing.T) {

	qs := sous.NewR11nQueueSet()

	r11n := &sous.Rectification{
		Pair: sous.DeployablePair{},
	}
	did := sous.DeploymentID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: "repo-one",
			},
		},
		Cluster: "cluster-one",
	}
	r11n.Pair.SetID(did)

	_, ok := qs.PushIfEmpty(r11n)
	if !ok {
		t.Fatal("precondition failed: failed to push r11n")
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

	const wantLen = 1
	gotLen := len(dr.Deployments)
	if gotLen != wantLen {
		t.Fatalf("got %d queued deployments; want %d", gotLen, wantLen)
	}

	const wantCount = 1
	gotCount := dr.Deployments[did]
	if gotCount != wantCount {
		t.Errorf("got %d queued rectifications for %q; want %d", gotCount, did, wantCount)
	}
}
