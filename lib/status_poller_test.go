package sous

import (
	"fmt"
	"testing"

	"github.com/samsalisbury/semv"
)

func TestSubPoller_ComputeState(t *testing.T) {
	testRepo := "github.com/opentable/example"
	testDir := "test"
	rezErr := fmt.Errorf("something bad")

	versionDep := func(version string) *Deployment {
		return &Deployment{
			Cluster: &Cluster{},
			SourceID: SourceID{
				Location: SourceLocation{
					Repo: testRepo,
					Dir:  testDir,
				},
				Version: semv.MustParse(version),
			},
		}
	}

	diffRez := func(desc string, err error) *DiffResolution {
		return &DiffResolution{
			Desc:  desc,
			Error: err,
		}
	}

	testCompute := func(version string, intent *Deployment, stable, current *DiffResolution, expected ResolveState) {
		sub := subPoller{
			idFilter: &ResolveFilter{
				Tag: version,
			},
		}
		if actual := sub.computeState(intent, stable, current); expected != actual {
			t.Errorf("sub.computeState(%v, %v, %v) -> %v != %v", intent, stable, current, actual, expected)
		}
	}

	testCompute("1.0", nil, nil, nil, ResolveNotStarted)
	testCompute("1.0", versionDep("0.9"), nil, nil, ResolveNotVersion)
	testCompute("1.0", versionDep("1.0"), nil, nil, ResolvePendingRequest)

	testCompute("1.0", versionDep("1.0"), diffRez("update", nil), nil, ResolveInProgress) //known update , no outcome yet
	testCompute("1.0", versionDep("1.0"), nil, diffRez("update", nil), ResolveInProgress) //new update   , now in progress
	testCompute("1.0", versionDep("1.0"), diffRez("unchanged", nil), diffRez("update", nil), ResolveInProgress)
	testCompute("1.0", versionDep("1.0"), diffRez("create", rezErr), diffRez("update", nil), ResolveInProgress)

	testCompute("1.0", versionDep("1.0"), diffRez("unchanged", rezErr), nil, ResolveErred)
	testCompute("1.0", versionDep("1.0"), nil, diffRez("unchanged", rezErr), ResolveErred)
	testCompute("1.0", versionDep("1.0"), diffRez("unchanged", nil), diffRez("unchanged", rezErr), ResolveErred)
	testCompute("1.0", versionDep("1.0"), diffRez("create", rezErr), diffRez("unchanged", rezErr), ResolveErred)

	testCompute("1.0", versionDep("1.0"), diffRez("unchanged", nil), nil, ResolveComplete)
	testCompute("1.0", versionDep("1.0"), nil, diffRez("unchanged", nil), ResolveComplete)
	testCompute("1.0", versionDep("1.0"), diffRez("create", rezErr), diffRez("unchanged", nil), ResolveComplete)
}
