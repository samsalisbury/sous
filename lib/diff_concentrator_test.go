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

	makeDepl := func(repo, verstr string, num int) *Deployment {
		version := semv.MustParse(verstr)
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

	repoOne := "github.com/opentable/one"
	repoTwo := "github.com/opentable/two"
	repoThree := "github.com/opentable/three"
	repoFour := "github.com/opentable/four"
	repoFive := "github.com/opentable/five"

	existing.MustAdd(makeDepl(repoOne, "111.1.1", 1)) //remove
	//intended: gone

	existing.MustAdd(makeDepl(repoTwo, "1.0.0", 1)) //same
	intended.MustAdd(makeDepl(repoTwo, "1.0.0", 1)) //same

	// existing: doesn't yet
	intended.MustAdd(makeDepl(repoFour, "1.0.0", 1)) //create

	existing.MustAdd(makeDepl(repoThree, "1.0.0", 1)) //changed
	intended.MustAdd(makeDepl(repoThree, "1.0.0", 2)) //changed

	existing.MustAdd(makeDepl(repoFive, "1.0.0", 1)) //changed
	intended.MustAdd(makeDepl(repoFive, "2.0.0", 1)) //changed

	dc := existing.Diff(intended).Concentrate(defs)
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

	if assert.Len(ds.Changed, 2, "Should have two modified items.") {
		chNum, chVer := ds.Changed[0], ds.Changed[1]
		if repoThree == chVer.name.Source.Repo {
			chNum, chVer = chVer, chNum
		}
		assert.Equal(repoThree, string(chNum.name.Source.Repo))
		assert.Equal(repoThree, string(chNum.Prior.Source.Repo))
		assert.Equal(repoThree, string(chNum.Post.Source.Repo))
		/*
			log.Printf("%+v", chNum)
			log.Printf("%+v", chNum.Prior)
			log.Printf("%+v", chNum.Post)
		*/
		assert.Equal(chNum.Prior.Deployments["test"].NumInstances, 1)
		assert.Equal(chNum.Post.Deployments["test"].NumInstances, 2)

		assert.Equal(repoFive, string(chVer.name.Source.Repo))
		assert.Equal(repoFive, string(chVer.Prior.Source.Repo))
		assert.Equal(repoFive, string(chVer.Post.Source.Repo))
		ver1 := semv.MustParse("1.0.0")
		ver2 := semv.MustParse("2.0.0")
		assert.Equal(ver1, chVer.Prior.Deployments["test"].Version)
		assert.Equal(ver2, chVer.Post.Deployments["test"].Version)
	}

	if assert.Equal(ds.New.Len(), 1, "Should have one added item.") {
		it, _ := ds.New.Any(func(*Manifest) bool { return true })
		assert.Equal(string(it.Source.Repo), repoFour)
	}

}
