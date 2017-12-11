package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestEmptyDiff(t *testing.T) {

	intended := NewDeployments()
	existing := NewDeployments()

	dc := intended.Diff(existing)
	ds := dc.collect()

	if len(ds.Pairs) != 0 {
		t.Errorf("got %d pairs; want 0", len(ds.Pairs))
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

func gone(ds diffSet) []*DeployablePair {
	return ds.Filter(func(dp *DeployablePair) bool {
		return dp.Kind == RemovedKind
	})
}

func same(ds diffSet) []*DeployablePair {
	return ds.Filter(func(dp *DeployablePair) bool {
		return dp.Kind == SameKind
	})
}

func changed(ds diffSet) []*DeployablePair {
	return ds.Filter(func(dp *DeployablePair) bool {
		return dp.Kind == ModifiedKind
	})
}

func created(ds diffSet) []*DeployablePair {
	return ds.Filter(func(dp *DeployablePair) bool {
		return dp.Kind == AddedKind
	})
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

	if assert.Len(gone(ds), 1, "Should have one deleted item.") {
		it := gone(ds)[0]
		assert.Equal(string(it.Prior.SourceID.Location.Repo), repoOne)
	}

	if assert.Len(same(ds), 1, "Should have one unchanged item.") {
		it := same(ds)[0]
		assert.Equal(string(it.Post.SourceID.Location.Repo), repoTwo)
	}

	ch := changed(ds)
	if assert.Len(ch, 1, "Should have one modified item.") {
		assert.Equal(repoThree, string(ch[0].name.ManifestID.Source.Repo))
		assert.Equal(repoThree, string(ch[0].Prior.SourceID.Location.Repo))
		assert.Equal(repoThree, string(ch[0].Post.SourceID.Location.Repo))
		assert.Equal(ch[0].Post.NumInstances, 1)
		assert.Equal(ch[0].Prior.NumInstances, 2)
	}

	if assert.Len(created(ds), 1, "Should have one added item.") {
		it := created(ds)[0]
		assert.Equal(string(it.Post.SourceID.Location.Repo), repoFour)
	}
}

func makeDeplState(repo string, num int, st DeployStatus, data interface{}) *DeployState {
	return &DeployState{
		ExecutorData: data,
		Deployment:   *makeDepl(repo, num),
		Status:       st,
	}
}

func testStateDiff(exists *Deployment, intend *DeployState) diffSet {
	intended := NewDeployStates()
	existing := NewDeployments()

	if exists != nil {
		existing.MustAdd(exists)
	}

	if intend != nil {
		intended.MustAdd(intend)
	}

	return intended.Diff(existing).collect()
}

func TestRealStateDiff(t *testing.T) {
	assert := assert.New(t)

	assertLengths := func(msg string, set diffSet, goneLen, sameLen, changedLen, createLen int) {
		assert.Len(gone(set), goneLen, "Checking Gone for %s", msg)
		assert.Len(same(set), sameLen, "Checking Same for %s", msg)
		assert.Len(changed(set), changedLen, "Checking Changed for %s", msg)
		assert.Len(created(set), createLen, "Checking New for %s", msg)
	}

	assertGone := func(set diffSet) { assertLengths("gone", set, 1, 0, 0, 0) }
	assertSame := func(set diffSet) { assertLengths("same", set, 0, 1, 0, 0) }
	assertChanged := func(set diffSet) { assertLengths("changed", set, 0, 0, 1, 0) }
	assertCreated := func(set diffSet) { assertLengths("created", set, 0, 0, 0, 1) }

	repo := "https://github.com/opentable/one"

	set := testStateDiff(nil, makeDeplState(repo, 1, DeployStatusActive, `gone`)) //remove
	assertGone(set)
	assert.Equal(gone(set)[0].ExecutorData, `gone`)

	set = testStateDiff(makeDepl(repo, 1), makeDeplState(repo, 1, DeployStatusActive, `same`))
	assertSame(set)
	assert.Equal(same(set)[0].ExecutorData, `same`)

	set = testStateDiff(makeDepl(repo, 1), makeDeplState(repo, 1, DeployStatusPending, `changed-pending`))
	assertChanged(set)
	assert.Equal(changed(set)[0].ExecutorData, `changed-pending`)

	set = testStateDiff(makeDepl(repo, 1), makeDeplState(repo, 1, DeployStatusFailed, `changed-failed`))
	assertChanged(set)
	assert.Equal(changed(set)[0].ExecutorData, `changed-failed`)

	set = testStateDiff(makeDepl(repo, 1), makeDeplState(repo, 2, DeployStatusActive, `changed-active`))
	assertChanged(set)
	assert.Equal(changed(set)[0].ExecutorData, `changed-active`)

	set = testStateDiff(makeDepl(repo, 1), nil)
	assertCreated(set)
	assert.Zero(created(set)[0].ExecutorData)
}
