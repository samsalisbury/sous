package sous

import (
	"log"
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/nyarly/testify/require"
	"github.com/samsalisbury/semv"
)

func TestEmptyDiffConcentration(t *testing.T) {

	intended := NewDeployments()
	existing := NewDeployments()

	dc := intended.Diff(existing)
	ds := dc.collect()

	if ds.New.Len() != 0 {
		t.Errorf("got %d new; want 0", ds.New.Len())
	}
	if ds.Gone.Len() != 0 {
		t.Errorf("got %d gone; want 0", ds.Gone.Len())
	}
	if ds.Same.Len() != 0 {
		t.Errorf("got %d same; want 0", ds.Same.Len())
	}
	if len(ds.Changed) != 0 {
		t.Errorf("got %d changed; want 0", len(ds.Changed))
	}
}

func TestRealDiffConcentration(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	assert := assert.New(t)
	require := require.New(t)

	intended := NewDeployments()
	existing := NewDeployments()
	defs := Defs{Clusters: map[string]*Cluster{"test": &Cluster{}}}

	makeDepl := func(repo string, num int) *Deployment {
		version, _ := semv.Parse("1.1.1-latest")
		cl := defs.Clusters["test"]
		owners := OwnerSet{}
		owners.Add("judson")
		return &Deployment{
			SourceID: SourceID{
				Location: SourceLocation{
					Repo: repo,
				},
				Version: version,
			},
			Cluster:     cl,
			ClusterName: "test",
			DeployConfig: DeployConfig{
				NumInstances: num,
				Env:          map[string]string{},
				Resources: map[string]string{
					"cpu":    ".1",
					"memory": "100",
					"ports":  "1",
				},
			},
			Owners: owners,
		}
	}

	repoOne := "https://github.com/opentable/one"
	repoTwo := "https://github.com/opentable/two"
	repoThree := "https://github.com/opentable/three"
	repoFour := "https://github.com/opentable/four"

	intended.MustAdd(makeDepl(repoOne, 1)) //remove

	existing.MustAdd(makeDepl(repoTwo, 1)) //same
	intended.MustAdd(makeDepl(repoTwo, 1)) //same

	existing.MustAdd(makeDepl(repoThree, 1)) //changed
	intended.MustAdd(makeDepl(repoThree, 2)) //changed

	existing.MustAdd(makeDepl(repoFour, 1)) //create

	dc := intended.Diff(existing).Concentrate(defs)
	ds, err := dc.collect()
	require.NoError(err)

	if assert.Len(ds.Gone.Snapshot(), 1, "Should have one deleted item.") {
		it, _ := ds.Gone.Any(func(*Manifest) bool { return true })
		assert.Equal(string(it.Source.Repo), repoOne)
	}

	if assert.Len(ds.Same.Snapshot(), 1, "Should have one unchanged item.") {
		it, _ := ds.Same.Any(func(*Manifest) bool { return true })
		assert.Equal(string(it.Source.Repo), repoTwo)
	}

	if assert.Len(ds.Changed, 1, "Should have one modified item.") {
		assert.Equal(repoThree, string(ds.Changed[0].name.Source.Repo))
		assert.Equal(repoThree, string(ds.Changed[0].Prior.Source.Repo))
		assert.Equal(repoThree, string(ds.Changed[0].Post.Source.Repo))
		assert.Equal(ds.Changed[0].Post.Deployments["test"].NumInstances, 1)
		assert.Equal(ds.Changed[0].Prior.Deployments["test"].NumInstances, 2)
	}

	if assert.Equal(ds.New.Len(), 1, "Should have one added item.") {
		it, _ := ds.New.Any(func(*Manifest) bool { return true })
		assert.Equal(string(it.Source.Repo), repoFour)
	}

}
