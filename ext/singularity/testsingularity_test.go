package singularity

import (
	"log"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
)

// A testSingularity represents a test singularity instance.
//
// It provides functions that make it easy to construct a consistent
// milieu in which tests can be run. The strategy for writing tests
// with this is to construct a healthy and consistent world, and then
// to introduce specific flaws against which tests can be written.
type testSingularity struct {
	Parent   *testFixture
	BaseURL  string
	Requests map[string]*testRequestParent
}

// AddCluster adds a cluster and ensures a singularity exists for its baseURL.
// It creates the necessary singularity if it doesn't exist.
//
// It returns the singularity with the same base url.
func (ts *testSingularity) AddCluster(name string) {
	if ts.Parent.Clusters == nil {
		ts.Parent.Clusters = sous.Clusters{}
	}
	cluster := &sous.Cluster{Name: name, BaseURL: ts.BaseURL}
	ts.Parent.Clusters[name] = cluster
}

// AddRequest adds a new RequestParent and associated request to the test
// fixture. The request parent created is identical to what defaultRequestParent
// returns, except the ID is set to requestID. The configure func is passed this
// request parent and may modify it before AddRequest returns it wrapped.
//
// It barfs if the requestID is not parseable with ParseRequestID, or if
// it ends up with an empty Cluster or SourceLocation, or if the requestID is
// not unique within this testSingularity, or if the cluster implied by the
// request ID is not already defined.
func (ts *testSingularity) AddRequestParent(requestID string, configure func(*dtos.SingularityRequestParent)) *testRequestParent {
	deployID, err := ParseRequestID(requestID)
	if err != nil {
		log.Panicf("Error parsing requestID: %s", err)
	}
	if deployID.Cluster == "" {
		log.Panicf("Request ID %q has an empty cluster component.", requestID)
	}
	if _, ok := ts.Parent.Clusters[deployID.Cluster]; !ok {
		log.Panicf("Cluster %q not defined (from request id %q)", deployID.Cluster, requestID)
	}
	if deployID.ManifestID.Source.Repo == "" {
		log.Panicf("Request ID %q has an empty source repo component.", requestID)
	}

	requestParent := newSingularityRequestParent(newRequestParentParams{
		requestID: requestID,
	})

	if configure != nil {
		configure(requestParent)
	}
	if ts.Requests == nil {
		ts.Requests = map[string]*testRequestParent{}
	}
	if _, exists := ts.Requests[requestParent.Request.Id]; exists {
		log.Panicf("request with ID %q already added", requestParent.Request.Id)
	}

	testRequestParent := &testRequestParent{
		Parent:        ts,
		RequestParent: requestParent,
	}
	ts.Requests[requestParent.Request.Id] = testRequestParent
	return testRequestParent
}
