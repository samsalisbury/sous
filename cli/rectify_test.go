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
	rs := []sous.RepoURL{"github.com/ot/one", "github.com/ot/two"}
	os := []sous.RepoOffset{"up", "down"}

	for _, c := range cs {
		for _, r := range rs {
			for _, o := range os {
				ds = append(ds, &sous.Deployment{
					ClusterNickname: c,
					SourceID: sous.SourceID{
						RepoURL:    r,
						RepoOffset: o,
					},
				})
			}
		}
	}

	//	for i, d := range ds {
	//		fmt.Printf("%d: %#v\n", i, d)
	//	}
	//
	f := rectifyFlags{}
	assert.Nil(f.buildPredicate())

	f.repo = string(rs[0])
	pd := f.buildPredicate()
	assert.NotNil(pd)
	filtered := filter(ds, pd)
	assert.Contains(filtered, ds[0])
	assert.Contains(filtered, ds[1])
	assert.Contains(filtered, ds[4])
	assert.Contains(filtered, ds[5])
	assert.Len(filtered, 4)

	f.offset = string(os[0])
	pd = f.buildPredicate()
	assert.NotNil(pd)
	filtered = filter(ds, pd)
	assert.Contains(filtered, ds[0])
	assert.Contains(filtered, ds[4])
	assert.Len(filtered, 2)

	f.cluster = cs[0]
	pd = f.buildPredicate()
	assert.NotNil(pd)
	filtered = filter(ds, pd)
	assert.Contains(filtered, ds[0])
	assert.Len(filtered, 1)

	f = rectifyFlags{cluster: cs[1]}
	pd = f.buildPredicate()
	assert.NotNil(pd)
	filtered = filter(ds, pd)
	assert.Contains(filtered, ds[4])
	assert.Contains(filtered, ds[5])
	assert.Contains(filtered, ds[6])
	assert.Contains(filtered, ds[7])
	assert.Len(filtered, 4)
}

func filter(ds []*sous.Deployment, pd sous.DeploymentPredicate) (fd []*sous.Deployment) {
	for _, d := range ds {
		if pd(d) {
			fd = append(fd, d)
		}
	}
	return

}
