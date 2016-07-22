package sous

import (
	"log"
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestEmptyDiff(t *testing.T) {
	assert := assert.New(t)

	intended := make(Deployments, 0)
	existing := make(Deployments, 0)

	dc := intended.Diff(existing)
	ds := dc.collect()

	assert.Len(ds.New, 0)
	assert.Len(ds.Gone, 0)
	assert.Len(ds.Same, 0)
	assert.Len(ds.Changed, 0)
}

func makeDepl(repo string, num int) *Deployment {
	version, _ := semv.Parse("1.1.1-latest")
	owners := OwnerSet{}
	owners.Add("judson")
	return &Deployment{
		SourceID: SourceID{
			RepoURL:    RepoURL(repo),
			Version:    version,
			RepoOffset: "",
		},
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

func TestRealDiff(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	assert := assert.New(t)

	intended := make(Deployments, 0)
	existing := make(Deployments, 0)

	repoOne := "https://github.com/opentable/one"
	repoTwo := "https://github.com/opentable/two"
	repoThree := "https://github.com/opentable/three"
	repoFour := "https://github.com/opentable/four"

	intended.Add(makeDepl(repoOne, 1)) //remove

	existing.Add(makeDepl(repoTwo, 1)) //same
	intended.Add(makeDepl(repoTwo, 1)) //same

	existing.Add(makeDepl(repoThree, 1)) //changed
	intended.Add(makeDepl(repoThree, 2)) //changed

	existing.Add(makeDepl(repoFour, 1)) //create

	dc := intended.Diff(existing)
	ds := dc.collect()

	if assert.Len(ds.Gone, 1, "Should have one deleted item.") {
		assert.Equal(string(ds.Gone[0].SourceID.RepoURL), repoOne)
	}

	if assert.Len(ds.Same, 1, "Should have one unchanged item.") {
		assert.Equal(string(ds.Same[0].SourceID.RepoURL), repoTwo)
	}

	if assert.Len(ds.Changed, 1, "Should have one modified item.") {
		assert.Equal(repoThree, string(ds.Changed[0].name.source.RepoURL))
		assert.Equal(repoThree, string(ds.Changed[0].Prior.SourceID.RepoURL))
		assert.Equal(repoThree, string(ds.Changed[0].Post.SourceID.RepoURL))
		assert.Equal(ds.Changed[0].Post.NumInstances, 1)
		assert.Equal(ds.Changed[0].Prior.NumInstances, 2)
	}

	if assert.Len(ds.New, 1, "Should have one added item.") {
		assert.Equal(string(ds.New[0].SourceID.RepoURL), repoFour)
	}

}
