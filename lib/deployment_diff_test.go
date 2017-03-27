package sous

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/samsalisbury/semv"
)

func TestEmptyDiff(t *testing.T) {

	intended := NewDeployments()
	existing := NewDeployments()

	dc := intended.Diff(existing)
	ds := dc.collect()

	if len(ds.New) != 0 {
		t.Errorf("got %d new; want 0", len(ds.New))
	}
	if len(ds.Gone) != 0 {
		t.Errorf("got %d gone; want 0", len(ds.Gone))
	}
	if len(ds.Same) != 0 {
		t.Errorf("got %d same; want 0", len(ds.Same))
	}
	if len(ds.Changed) != 0 {
		t.Errorf("got %d changed; want 0", len(ds.Changed))
	}
}

func makeDepl(repo string, num int) *Deployment {
	version, _ := semv.Parse("1.1.1-latest")
	owners := OwnerSet{}
	owners.Add("judson")
	return &Deployment{
		SourceID: SourceID{
			Location: SourceLocation{
				Repo: repo,
			},
			Version: version,
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
	assert := assert.New(t)

	intended := NewDeployments()
	existing := NewDeployments()

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

	dc := intended.Diff(existing)
	ds := dc.collect()

	if assert.Len(ds.Gone, 1, "Should have one deleted item.") {
		it := ds.Gone[0]
		assert.Equal(string(it.Prior.SourceID.Location.Repo), repoOne)
	}

	if assert.Len(ds.Same, 1, "Should have one unchanged item.") {
		it := ds.Same[0]
		assert.Equal(string(it.Post.SourceID.Location.Repo), repoTwo)
	}

	if assert.Len(ds.Changed, 1, "Should have one modified item.") {
		assert.Equal(repoThree, string(ds.Changed[0].name.ManifestID.Source.Repo))
		assert.Equal(repoThree, string(ds.Changed[0].Prior.SourceID.Location.Repo))
		assert.Equal(repoThree, string(ds.Changed[0].Post.SourceID.Location.Repo))
		assert.Equal(ds.Changed[0].Post.NumInstances, 1)
		assert.Equal(ds.Changed[0].Prior.NumInstances, 2)
	}

	if assert.Len(ds.New, 1, "Should have one added item.") {
		it := ds.New[0]
		assert.Equal(string(it.Post.SourceID.Location.Repo), repoFour)
	}
}

func makeDeplState(repo string, num int, st DeployStatus) *DeployState {
	return &DeployState{
		Deployment: *makeDepl(repo, num),
		Status:     st,
	}
}

func TestRealStateDiff(t *testing.T) {
	assert := assert.New(t)

	intended := NewDeployStates()
	existing := NewDeployments()

	repoOne := "https://github.com/opentable/one"
	repoTwo := "https://github.com/opentable/two"
	repoTwoA := "https://github.com/opentable/two-a"
	repoThree := "https://github.com/opentable/three"
	repoFour := "https://github.com/opentable/four"
	repoFive := "https://github.com/opentable/five"

	intended.MustAdd(makeDeplState(repoOne, 1, DeployStatusActive)) //remove

	existing.MustAdd(makeDepl(repoTwo, 1))                          //same
	intended.MustAdd(makeDeplState(repoTwo, 1, DeployStatusActive)) //same

	existing.MustAdd(makeDepl(repoTwoA, 1))                           //same
	intended.MustAdd(makeDeplState(repoTwoA, 1, DeployStatusPending)) //same

	existing.MustAdd(makeDepl(repoFive, 1))                          //changed
	intended.MustAdd(makeDeplState(repoFive, 1, DeployStatusFailed)) //changed

	existing.MustAdd(makeDepl(repoThree, 1))                          //changed
	intended.MustAdd(makeDeplState(repoThree, 2, DeployStatusActive)) //changed

	existing.MustAdd(makeDepl(repoFour, 1)) //create

	dc := intended.Diff(existing)
	ds := dc.collect()

	if assert.Len(ds.Gone, 1, "Should have one deleted item.") {
		it := ds.Gone[0]
		assert.Equal(string(it.Prior.SourceID.Location.Repo), repoOne)
	}

	if assert.Len(ds.Same, 2, "Should have two unchanged items.") {
		it := ds.Same[0]
		pedActive := ds.Same[1]
		if it.Status == DeployStatusPending {
			it = ds.Same[1]
			pedActive = ds.Same[0]
		}
		assert.Equal(string(it.Post.SourceID.Location.Repo), repoTwo)
		assert.Equal(string(pedActive.Post.SourceID.Location.Repo), repoTwoA)
		assert.Equal(pedActive.Status, DeployStatusPending)
	}

	if assert.Len(ds.Changed, 2, "Should have two modified items.") {
		three := ds.Changed[0]
		five := ds.Changed[1]
		if ds.Changed[0].Prior.SourceID.Location.Repo == repoFive {
			three = ds.Changed[1]
			five = ds.Changed[0]
		}

		assert.Equal(repoThree, string(three.name.ManifestID.Source.Repo))
		assert.Equal(repoThree, string(three.Prior.SourceID.Location.Repo))
		assert.Equal(repoThree, string(three.Post.SourceID.Location.Repo))
		assert.Equal(three.Post.NumInstances, 1)
		assert.Equal(three.Prior.NumInstances, 2)

		assert.Equal(repoFive, string(five.name.ManifestID.Source.Repo))
		assert.Equal(repoFive, string(five.Prior.SourceID.Location.Repo))
		assert.Equal(repoFive, string(five.Post.SourceID.Location.Repo))
		assert.Equal(five.Post.NumInstances, 1)
		assert.Equal(five.Prior.NumInstances, 1)
		assert.Equal(five.Status, DeployStatusFailed)
	}

	if assert.Len(ds.New, 1, "Should have one added item.") {
		it := ds.New[0]
		assert.Equal(string(it.Post.SourceID.Location.Repo), repoFour)
	}
}
