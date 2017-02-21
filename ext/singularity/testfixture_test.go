package singularity

import (
	"crypto/sha1"
	"fmt"
	"log"
	"path/filepath"
	"runtime"

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
	Registry      *testRegistry
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
	// Error to be returned instead of RequestParent.
	Error   error
	Deploys map[string]*testDeploy
}

// A testDeploy represents a single deployment.
type testDeploy struct {
	Parent            *testRequest
	DeployHistoryItem *dtos.SingularityDeployHistory
}

func (tf *testFixture) DeployReaderFactory(c *sous.Cluster) DeployReader {
	return &testDeployReader{Fixture: tf}
}

type testDeployReader struct {
	Fixture *testFixture
}

func (tdr *testDeployReader) GetRequests() (dtos.SingularityRequestParentList, error) {
	rpl := dtos.SingularityRequestParentList{}
	for _, singularity := range tdr.Fixture.Singularities {
		for _, request := range singularity.Requests {
			if request.Error != nil {
				return nil, request.Error
			}
			rpl = append(rpl, request.RequestParent)
		}
	}
	return rpl, nil
}

func (tdr *testDeployReader) GetRequest(requestID string) (*dtos.SingularityRequestParent, error) {
	request, err := tdr.GetTestRequest(requestID)
	if err != nil {
		return nil, httpErr(404, err.Error())
	}
	if request.RequestParent == nil {
		log.Panicf("testRequest has no RequestParent")
	}
	return request.RequestParent, nil
}

func (tdr *testDeployReader) GetTestRequest(requestID string) (*testRequest, error) {
	did, err := ParseRequestID(requestID)
	if err != nil {
		log.Panic(err)
	}
	// Let these panic if there is nothing there.
	cluster, ok := tdr.Fixture.Clusters[did.Cluster]
	if !ok {
		log.Panicf("No cluster called %q", did.Cluster)
	}
	baseURL := cluster.BaseURL
	singularity, ok := tdr.Fixture.Singularities[baseURL]
	if !ok {
		log.Panicf("No Singularity for base URL %q (of cluster %q)", baseURL, did.Cluster)
	}
	request, ok := singularity.Requests[requestID]
	if !ok {
		return nil, fmt.Errorf("no request named %q in the fixture", requestID)
	}
	return request, nil
}

type httpError struct {
	Code int
	Text string
}

func (h *httpError) Error() string   { return fmt.Sprintf("HTTP %d: %s", h.Code, h.Text) }
func (h *httpError) Temporary() bool { return true }

func httpErr(code int, format string, a ...interface{}) error {
	err := &httpError{Code: 404, Text: fmt.Sprintf(format, a...)}
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Panicf("httpErr unable to get its caller")
	}
	file = filepath.Base(file)
	log.Printf("%s:%d: %s", file, line, err)
	return err
}

func (tdr *testDeployReader) GetDeploy(requestID, deployID string) (*dtos.SingularityDeployHistory, error) {
	if deployID == "" {
		log.Panic("GetDeploy passed an empty deployID")
	}
	request, err := tdr.GetTestRequest(requestID)
	if err != nil {
		// TODO: Find out what Swaggering does and ensure we are emulating that.
		return nil, httpErr(404, "no deploy %q; no request named %q in the fixture", deployID, requestID)
	}
	deploy, ok := request.Deploys[deployID]
	if !ok {
		return nil, httpErr(404, "no deploy %q in request %q", deployID, requestID)
	}
	return deploy.DeployHistoryItem, nil
}

type testRegistry struct {
	Images map[string]*testImage
}

func (tr *testRegistry) GetArtifact(sid sous.SourceID) (*sous.BuildArtifact, error) {
	panic("implements sous.Registry")
}

func (tr *testRegistry) GetSourceID(ba *sous.BuildArtifact) (sous.SourceID, error) {
	panic("implements sous.Registry")
}

func (tr *testRegistry) ImageLabels(imageName string) (map[string]string, error) {
	return tr.Images[imageName].labels, nil
}

func (tr *testRegistry) ListSourceIDs() ([]sous.SourceID, error) {
	panic("implements sous.Registry")
}

func (tr *testRegistry) Warmup(string) error {
	panic("implements sous.Registry")
}

type testImage struct {
	labels map[string]string
}

func newTestRegistry() *testRegistry {
	return &testRegistry{
		Images: map[string]*testImage{},
	}
}

// AddImage adds an image with name provided and with labels corresponding to
// repo, offset and tag.
func (tr *testRegistry) AddImage(name, repo, offset, tag string) {
	if offset != "" {
		offset = "," + offset
	}
	revision := string(sha1.New().Sum([]byte(name)))
	imageLabels := map[string]string{
		"com.opentable.sous.repo_url":    repo,
		"com.opentable.sous.version":     tag,
		"com.opentable.sous.revision":    revision,
		"com.opentable.sous.repo_offset": offset,
	}
	tr.Images[name] = &testImage{
		labels: imageLabels,
	}
}

func testImageName(repo, offset, tag string) string {
	return fmt.Sprintf("docker.mycompany.com/%s%s:%s", repo, offset, tag)
}

// AddCluster adds a cluster and ensures a singularity exists for its baseURL.
// It creates the necessary singularity if it doesn't exist.
//
// It returns the singularity with the same base url.
func (tf *testFixture) AddCluster(name, baseURL string) *testSingularity {
	if tf.Clusters == nil {
		tf.Clusters = sous.Clusters{}
	}
	tf.Clusters[name] = &sous.Cluster{Name: name, BaseURL: baseURL}
	return tf.AddSingularity(baseURL)
}

// AddSingularity adds a singularity if none exist for baseURL. It returns
// the one that already existed, or the new one created.
func (tf *testFixture) AddSingularity(baseURL string) *testSingularity {
	if tf.Singularities == nil {
		tf.Singularities = map[string]*testSingularity{}
	}
	if s, ok := tf.Singularities[baseURL]; ok {
		return s
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
// not unique within this testSingularity, or if the cluster implied by the
// request ID is not already defined.
func (ts *testSingularity) AddRequest(requestID string, configure func(*dtos.SingularityRequestParent)) *testRequest {
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
// AddDeploy also adds:
//   - A corresponding docker image to the test registry owned
//     by the ancestor testFixture (at Parent.Parent.Parent)
//   - A corresponding entry in SingularityRequestDeployState if the
//     status is Pending or Active after configure is called.
func (tr *testRequest) AddDeploy(deployID string, configure func(*dtos.SingularityDeployHistory)) *testDeploy {
	if tr.Deploys == nil {
		tr.Deploys = map[string]*testDeploy{}
	}
	requestID := tr.RequestParent.Request.Id
	deployHistory := defaultDeployHistoryItem(requestID, deployID)

	did, err := ParseRequestID(tr.RequestParent.Request.Id)
	if err != nil {
		log.Fatal(err)
	}
	repo := did.ManifestID.Source.Repo
	offset := did.ManifestID.Source.Dir
	tag := "1.0.0"

	imageName := testImageName(repo, offset, tag)
	deployHistory.Deploy.ContainerInfo.Docker.Image = imageName

	// Add docker image to the test registry.
	tr.Parent.Parent.Registry.AddImage(imageName, repo, offset, tag)

	// All defaults are set, now pass the deploy to provided configure func.
	if configure != nil {
		configure(deployHistory)
	}
	// After this we can respond to the final value.

	deployMarker := &dtos.SingularityDeployMarker{
		User:      "some user",
		RequestId: tr.RequestParent.Request.Id,
		Message:   "some message",
		Timestamp: 0, // TODO: Maybe have a counter to increment these.
		DeployId:  deployID,
	}

	tr.RequestParent.RequestDeployState = &dtos.SingularityRequestDeployState{}

	// Add an entry to SingularityRequestDeployState if we have Pending or
	// Active deploy.
	if deployHistory.DeployResult.DeployState == dtos.SingularityDeployResultDeployStateWAITING {
		tr.RequestParent.RequestDeployState.PendingDeploy = deployMarker
	}
	if deployHistory.DeployResult.DeployState == dtos.SingularityDeployResultDeployStateSUCCEEDED {
		tr.RequestParent.RequestDeployState.ActiveDeploy = deployMarker
	}

	deploy := &testDeploy{
		Parent:            tr,
		DeployHistoryItem: deployHistory,
	}
	// Add the deploy to this testRequest.
	tr.Deploys[deployHistory.Deploy.Id] = deploy
	return deploy
}
