package cli

import (
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/stretchr/testify/assert"
)

func TestPredicateBuilder(t *testing.T) {
	assert := assert.New(t)

	ds := make([]*sous.Deployment, 0, 8)
	cs := []string{"cluster1", "cluster2"}
	rs := []string{"github.com/ot/one", "github.com/ot/two"}
	os := []string{"up", "down"}

	for _, c := range cs {
		for _, r := range rs {
			for _, o := range os {
				ds = append(ds, &sous.Deployment{
					ClusterNickname: c,
					SourceID: sous.SourceID{
						Repo:   r,
						Offset: o,
					},
				})
			}
		}
	}

	//	for i, d := range ds {
	//		fmt.Printf("%d: %#v\n", i, d)
	//	}
	//
	f := DeployFilterFlags{}
	assert.Nil(f.buildPredicate())

	f.Repo = string(rs[0])
	pd := f.buildPredicate()
	assert.NotNil(pd)
	filtered := filter(ds, pd)
	assert.Contains(filtered, ds[0])
	assert.Contains(filtered, ds[1])
	assert.Contains(filtered, ds[4])
	assert.Contains(filtered, ds[5])
	assert.Len(filtered, 4)

	f.Offset = string(os[0])
	pd = f.buildPredicate()
	assert.NotNil(pd)
	filtered = filter(ds, pd)
	assert.Contains(filtered, ds[0])
	assert.Contains(filtered, ds[4])
	assert.Len(filtered, 2)

	f.Cluster = cs[0]
	pd = f.buildPredicate()
	assert.NotNil(pd)
	filtered = filter(ds, pd)
	assert.Contains(filtered, ds[0])
	assert.Len(filtered, 1)

	f = DeployFilterFlags{Cluster: cs[1]}
	pd = f.buildPredicate()
	assert.NotNil(pd)
	filtered = filter(ds, pd)
	assert.Contains(filtered, ds[4])
	assert.Contains(filtered, ds[5])
	assert.Contains(filtered, ds[6])
	assert.Contains(filtered, ds[7])
	assert.Len(filtered, 4)

	f = DeployFilterFlags{All: true}
	pd = f.buildPredicate()
	assert.NotNil(pd)
	filtered = filter(ds, pd)
	assert.Len(filtered, 8)
}

func filter(ds []*sous.Deployment, pd sous.DeploymentPredicate) (fd []*sous.Deployment) {
	for _, d := range ds {
		if pd(d) {
			fd = append(fd, d)
		}
	}
	return

}
