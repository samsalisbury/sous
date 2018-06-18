package sous

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/suite"
)

type NameResolveTestSuite struct {
	suite.Suite
	reg         *DummyRegistry
	testCluster *Cluster
	depChans    *DeployableChans
	diffChans   *DeployableChans
}

func (nrs *NameResolveTestSuite) makeTestDep() *Deployable {
	return &Deployable{
		Status: DeployStatusActive,
		Deployment: &Deployment{
			DeployConfig: DeployConfig{
				NumInstances: 12,
			},
			ClusterName: "test",
			Cluster:     nrs.testCluster,
			SourceID:    MustNewSourceID("gh.com", "offset", "0.0.2"),
			Flavor:      "",
			Owners:      OwnerSet{},
			Kind:        "service",
		},
	}
}

func (nrs *NameResolveTestSuite) makeTestDepPair(prior, post *Deployable) *DeployablePair {
	var id DeploymentID
	if prior != nil {
		id = prior.ID()
	}
	if post != nil {
		id = post.ID()
	}

	return &DeployablePair{
		name:  id,
		Prior: prior,
		Post:  post,
	}
}

func (nrs *NameResolveTestSuite) makeBuildArtifact() *BuildArtifact {
	return &BuildArtifact{
		Type:            "docker",
		DigestReference: "asdfasdf",
		Qualities:       []Quality{},
	}
}

func TestNameResolveSuite(t *testing.T) {
	suite.Run(t, new(NameResolveTestSuite))
}

func (nrs *NameResolveTestSuite) SetupTest() {
	nrs.testCluster = &Cluster{Name: "test"}

	nrs.reg = NewDummyRegistry()

	dc := NewDeployableChans(10)
	nrs.diffChans = dc
}

func (nrs *NameResolveTestSuite) TearDownTest() {
}

func (nrs *NameResolveTestSuite) TestResolveNameGood() {
	ls, _ := logging.NewLogSinkSpy()
	da, err := resolveName(nrs.reg, nrs.makeTestDep(), ls)
	nrs.NotNil(da)
	nrs.Nil(err)
}

func (nrs *NameResolveTestSuite) TestResolveNameBad() {
	nrs.reg.FeedArtifact(nil, fmt.Errorf("badness"))

	ls, _ := logging.NewLogSinkSpy()
	da, err := resolveName(nrs.reg, nrs.makeTestDep(), ls)
	nrs.Nil(da.BuildArtifact)
	nrs.Error(err.Error)
}

func (nrs *NameResolveTestSuite) TestResolveNameSkipped() {
	noInstances := nrs.makeTestDep()
	noInstances.DeployConfig.NumInstances = 0

	ls, _ := logging.NewLogSinkSpy()
	da, err := resolveName(nrs.reg, noInstances, ls)
	nrs.Nil(da.BuildArtifact)
	nrs.Nil(err)
}

func (nrs *NameResolveTestSuite) TestResolveNameStartChannel() {
	ls, _ := logging.NewLogSinkSpy()
	nrs.depChans = nrs.diffChans.ResolveNames(context.Background(), nrs.reg, ls)
	nrs.diffChans.Pairs <- nrs.makeTestDepPair(nil, nrs.makeTestDep())

	select {
	case started := <-nrs.depChans.Pairs:
		nrs.NotNil(started)
		nrs.NotNil(started.Post.Deployment)
		nrs.NotNil(started.Post.BuildArtifact)
	case err := <-nrs.depChans.Errs:
		nrs.Fail("Unexpected error: %v", err)
	case <-time.After(time.Second / 2):
		nrs.Fail("Timeout waiting for depChans to resolve")
	}
}

func (nrs *NameResolveTestSuite) TestResolveNameUpdateChannel() {
	ls, _ := logging.NewLogSinkSpy()
	nrs.depChans = nrs.diffChans.ResolveNames(context.Background(), nrs.reg, ls)

	pair := &DeployablePair{
		Prior: nrs.makeTestDep(),
		Post:  nrs.makeTestDep(),
	}

	pair.Post.NumInstances = pair.Prior.NumInstances + 3

	nrs.diffChans.Pairs <- pair

	select {
	case updated := <-nrs.depChans.Pairs:
		nrs.NotNil(updated)
		nrs.NotNil(updated.Post.BuildArtifact)
	case err := <-nrs.depChans.Errs:
		nrs.Fail("Unexpected error: %v", err)
	case <-time.After(time.Second / 2):
		nrs.Fail("Timeout waiting for depChans to resolve")
	}
}

func (nrs *NameResolveTestSuite) TestResolveNameStartChannelUnresolved() {
	nrs.reg.FeedArtifact(nil, fmt.Errorf("not found"))
	ls, _ := logging.NewLogSinkSpy()
	nrs.depChans = nrs.diffChans.ResolveNames(context.Background(), nrs.reg, ls)
	nrs.diffChans.Pairs <- nrs.makeTestDepPair(nil, nrs.makeTestDep())

	select {
	case <-nrs.depChans.Pairs:
		nrs.Fail("Shouldn't process a starting deployment")
	case err := <-nrs.depChans.Errs:
		nrs.Error(err.Error)
	case <-time.After(time.Second / 2):
		nrs.Fail("Timeout waiting for depChans to resolve")
	}
}

func (nrs *NameResolveTestSuite) TestResolveNameStopChannelUnresolved() {
	nrs.reg.FeedArtifact(nil, fmt.Errorf("not found"))

	ls, _ := logging.NewLogSinkSpy()
	nrs.depChans = nrs.diffChans.ResolveNames(context.Background(), nrs.reg, ls)
	nrs.diffChans.Pairs <- nrs.makeTestDepPair(nrs.makeTestDep(), nil)

	select {
	case stopped := <-nrs.depChans.Pairs:
		nrs.True(stopped.Kind() == RemovedKind, "got %s; want RemovedKind", stopped.Kind)
		nrs.NotNil(stopped)
	case err := <-nrs.depChans.Errs:
		nrs.Fail("Unexpected error: %v", err)
	case <-time.After(time.Second / 2):
		nrs.Fail("Timeout waiting for depChans to resolve")
	}
}

/*
func TestNameResolver(t *testing.T) {

	depChans := NewDeployableChans(1)
	diffChans := NewDeployableChans(1)

	reg := NewDummyRegistry()

	errChan := make(chan error)
	depChans.ResolveNames(reg, &diffChans, errChan)

	reg.FeedArtifact(makeBuildArtifact(), nil)
	reg.FeedArtifact(nil, fmt.Errorf("something wrong"))
	reg.FeedArtifact(makeBuildArtifact(), nil)
	reg.FeedArtifact(makeBuildArtifact(), nil)
	reg.FeedArtifact(makeBuildArtifact(), nil)
	reg.FeedArtifact(makeBuildArtifact(), nil)

	diffChans.Created <- makeTestDep()
	diffChans.Created <- makeTestDep()
	diffChans.Deleted <- makeTestDep()
	diffChans.Retained <- makeTestDep()
	diffChans.Modified <- &DeploymentPair{
		Prior: makeTestDep(),
		Post:  makeTestDep(),
	}

	diffChans.Close()
	time.Sleep(1 * time.Second)

	select {
	case cr := <-depChans.Start:
		assert.NotNil(cr)
		assert.NotNil(cr.Deployment)
		assert.NotNil(cr.BuildArtifact)
		assert.Empty(depChans.Start)
	default:
		assert.Fail("no created deployable")
	}

	select {
	case cr := <-depChans.Stop:
		assert.NotNil(cr)
		assert.NotNil(cr.Deployment)
		//assert.NotNil(cr.BuildArtifact) // stopped deployments don't need a BA
		assert.Empty(depChans.Stop)
	default:
		assert.Fail("no deleted deployable")
	}

	select {
	case cr := <-depChans.Stable:
		assert.NotNil(cr)
		assert.NotNil(cr.Deployment)
		//assert.NotNil(cr.BuildArtifact) // unchanged deployments don't need a BA
		assert.Empty(depChans.Stable)
	default:
		assert.Fail("no unchanged deployable")
	}

	select {
	case cr := <-depChans.Update:
		assert.NotNil(cr)
		assert.NotNil(cr.Prior.Deployment)
		assert.NotNil(cr.Prior.BuildArtifact)
		assert.NotNil(cr.Post.Deployment)
		assert.NotNil(cr.Post.BuildArtifact)
		assert.Empty(depChans.Update)
	default:
		assert.Fail("no changed deployable")
	}

	select {
	case err := <-errChan:
		assert.Error(err)
		assert.Empty(errChan)
	default:
		assert.Fail("no error")
	}
}
*/
