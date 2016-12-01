package sous

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type NameResolveTestSuite struct {
	suite.Suite
	reg         *DummyRegistry
	testCluster *Cluster
	depChans    *DeployableChans
	diffChans   *DiffChans
	errChan     chan error
}

func (nrs *NameResolveTestSuite) makeTestDep() *Deployment {
	return &Deployment{
		DeployConfig: DeployConfig{
			NumInstances: 12,
		},
		ClusterName: "test",
		Cluster:     nrs.testCluster,
		SourceID:    MustNewSourceID("gh.com", "offset", "0.0.2"),
		Flavor:      "",
		Owners:      OwnerSet{},
		Kind:        "service",
		Volumes:     Volumes{},
		Annotation:  Annotation{},
	}
}

func (nrs *NameResolveTestSuite) makeBuildArtifact() *BuildArtifact {
	return &BuildArtifact{
		Type:      "docker",
		Name:      "asdfasdf",
		Qualities: []Quality{},
	}
}

func TestNameResolveSuite(t *testing.T) {
	suite.Run(t, new(NameResolveTestSuite))
}

func (nrs *NameResolveTestSuite) SetupTest() {
	Log.Debug.SetOutput(os.Stderr)
	Log.Vomit.SetOutput(os.Stderr)
	Log.Warn.SetOutput(os.Stderr)

	nrs.testCluster = &Cluster{Name: "test"}

	nrs.reg = NewDummyRegistry()

	nrs.depChans = NewDeployableChans(10)
	dc := NewDiffChans(10)
	nrs.diffChans = &dc
	nrs.errChan = make(chan error, 10)
}

func (nrs *NameResolveTestSuite) TearDownTest() {
	Log.Debug.SetOutput(ioutil.Discard)
	Log.Vomit.SetOutput(ioutil.Discard)
	Log.Warn.SetOutput(ioutil.Discard)

}

func (nrs *NameResolveTestSuite) TestResolveNameGood() {
	da, err := resolveName(nrs.reg, nrs.makeTestDep())
	nrs.NotNil(da)
	nrs.NoError(err)
}

func (nrs *NameResolveTestSuite) TestResolveNameBad() {
	nrs.reg.FeedArtifact(nil, fmt.Errorf("badness"))

	da, err := resolveName(nrs.reg, nrs.makeTestDep())
	nrs.Nil(da.BuildArtifact)
	nrs.Error(err)
}

func (nrs *NameResolveTestSuite) TestResolveNameSkipped() {
	noInstances := nrs.makeTestDep()
	noInstances.DeployConfig.NumInstances = 0

	da, err := resolveName(nrs.reg, noInstances)
	nrs.Nil(da.BuildArtifact)
	nrs.NoError(err)
}

func (nrs *NameResolveTestSuite) TestResolveNameStartChannel() {
	nrs.depChans.ResolveNames(nrs.reg, nrs.diffChans, nrs.errChan)
	nrs.diffChans.Created <- nrs.makeTestDep()

	select {
	case started := <-nrs.depChans.Start:
		nrs.NotNil(started)
		nrs.NotNil(started.Deployment)
		nrs.NotNil(started.BuildArtifact)
	case err := <-nrs.errChan:
		nrs.Fail("Unexpected error: %v", err)
	case <-time.After(time.Second / 2):
		nrs.Fail("Timeout waiting for depChans to resolve")
	}
}

func (nrs *NameResolveTestSuite) TestResolveNameUpdateChannel() {
	nrs.depChans.ResolveNames(nrs.reg, nrs.diffChans, nrs.errChan)
	nrs.diffChans.Modified <- &DeploymentPair{
		Prior: nrs.makeTestDep(),
		Post:  nrs.makeTestDep(),
	}

	select {
	case updated := <-nrs.depChans.Update:
		nrs.NotNil(updated)
		nrs.NotNil(updated.Post.BuildArtifact)
	case err := <-nrs.errChan:
		nrs.Fail("Unexpected error: %v", err)
	case <-time.After(time.Second / 2):
		nrs.Fail("Timeout waiting for depChans to resolve")
	}
}

func (nrs *NameResolveTestSuite) TestResolveNameStartChannelUnresolved() {
	nrs.reg.FeedArtifact(nil, fmt.Errorf("not found"))
	nrs.depChans.ResolveNames(nrs.reg, nrs.diffChans, nrs.errChan)
	nrs.diffChans.Created <- nrs.makeTestDep()

	select {
	case <-nrs.depChans.Start:
		nrs.Fail("Shouldn't process a starting deployment")
	case err := <-nrs.errChan:
		nrs.Error(err)
	case <-time.After(time.Second / 2):
		nrs.Fail("Timeout waiting for depChans to resolve")
	}
}

func (nrs *NameResolveTestSuite) TestResolveNameStopChannelUnresolved() {
	nrs.reg.FeedArtifact(nil, fmt.Errorf("not found"))
	nrs.depChans.ResolveNames(nrs.reg, nrs.diffChans, nrs.errChan)
	nrs.diffChans.Deleted <- nrs.makeTestDep()

	select {
	case stopped := <-nrs.depChans.Stop:
		nrs.NotNil(stopped)
	case err := <-nrs.errChan:
		nrs.Fail("Unexpected error: %v", err)
	case <-time.After(time.Second / 2):
		nrs.Fail("Timeout waiting for depChans to resolve")
	}
}

/*
func TestNameResolver(t *testing.T) {

	depChans := NewDeployableChans(1)
	diffChans := NewDiffChans(1)

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
