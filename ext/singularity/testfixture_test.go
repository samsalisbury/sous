package singularity

import (
	"log"

	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
)

// A testFixture represents a state of the world for tests to run in.
//
// It provides functions that make it easy to construct a consistent
// milieu in which tests can be run. The strategy for writing tests
// with this is to construct a healthy and consistent world, and then
// to introduce specific flaws against which tests can be written.
type testFixture struct {
	Clusters      sous.Clusters
	Singularities map[string]*testSingularity
}

// A testSingularity represents a test singularity instance.
//
// It provides functions that make it easy to construct a consistent
// milieu in which tests can be run. The strategy for writing tests
// with this is to construct a healthy and consistent world, and then
// to introduce specific flaws against which tests can be written.
type testSingularity struct {
	Parent   *testFixture
	BaseURL  string
	Requests map[string]*testRequest
}

// A testRequest represents all the request-scoped data for a single
// singularity request.
//
// It provides functions that make it easy to construct a consistent
// milieu in which tests can be run. The strategy for writing tests
// with this is to construct a healthy and consistent world, and then
// to introduce specific flaws against which tests can be written.
type testRequest struct {
	Parent        *testSingularity
	RequestParent *dtos.SingularityRequestParent
	Deployments   map[string]*testDeploy
}

// A testDeploy represents a single deployment.
type testDeploy struct {
	Parent            *testRequest
	DeployHistoryItem *dtos.SingularityDeployHistory
}

// AddCluster adds a cluster and ensures a singularity exists for its baseURL.
// It creates the necessary singularity if it doesn't exist.
func (tf *testFixture) AddCluster(name, baseURL string) {
	if tf.Clusters == nil {
		tf.Clusters = sous.Clusters{}
	}
	tf.Clusters[name] = &sous.Cluster{Name: name, BaseURL: baseURL}
	tf.AddSingularity(baseURL)
}

func (tf *testFixture) AddSingularity(baseURL string) *testSingularity {
	if tf.Singularities == nil {
		tf.Singularities = map[string]*testSingularity{}
	}
	singularity := &testSingularity{
		Parent: tf,
	}
	tf.Singularities[baseURL] = singularity
	return singularity
}

// AddRequest adds a new RequestParent and associated request to the test
// fixture. The request parent created is identical to what defaultRequestParent
// returns, except the ID is set to requestID. The configure func is passed this
// request parent and may modify it before AddRequest returns it wrapped.
//
// It barfs if the requestID is not parseable with ParseRequestID, or if
// it ends up with an empty Cluster or SourceLocation, or if the requestID is
// not unique within this testSingularity.
func (ts *testSingularity) AddRequest(requestID string, configure func(*dtos.SingularityRequestParent)) *testRequest {
	deployID, err := ParseRequestID(requestID)
	if err != nil {
		log.Panicf("Error parsing requestID: %s", err)
	}
	if deployID.Cluster == "" {
		log.Panicf("Request ID %q has an empty cluster component.", requestID)
	}
	if deployID.ManifestID.Source.Repo == "" {
		log.Panicf("Request ID %q has an empty source repo component.", requestID)
	}
	parent := defaultRequestParent(requestID)
	if configure != nil {
		configure(parent)
	}
	if ts.Requests == nil {
		ts.Requests = map[string]*testRequest{}
	}
	request := &testRequest{
		Parent:        ts,
		RequestParent: parent,
	}
	if _, exists := ts.Requests[parent.Request.Id]; exists {
		log.Panicf("request with ID %q already added", parent.Request.Id)
	}
	ts.Requests[parent.Request.Id] = request
	return request
}

// AddDeploy adds a new DeployHistory linked with this request. The configure
// func is called on it to manipulate it before it's added to the deploy history
// and returned wrapped in a testDeploy.
//
// AddDeploy also adds a corresponding docker image to the test registry owned
// by the ancestor testFixture (at Parent.Parent.Parent).
func (tr *testRequest) AddDeploy(deployID string, configure func(*dtos.SingularityDeployHistory)) *testDeploy {
	if tr.Deployments == nil {
		tr.Deployments = map[string]*testDeploy{}
	}
	requestID := tr.RequestParent.Request.Id
	deployment := defaultDeployHistoryItem(requestID, deployID)
	if configure != nil {
		configure(deployment)
	}
	deploy := &testDeploy{
		Parent:            tr,
		DeployHistoryItem: deployment,
	}
	tr.Deployments[deployment.Deploy.Id] = deploy
	return deploy
}
