package sous

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

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

func TestStatusPoller(t *testing.T) {
	serversRE := regexp.MustCompile(`/servers$`)
	statusRE := regexp.MustCompile(`/status$`)
	var serversJSON, statusJSON []byte

	h := func(rw http.ResponseWriter, r *http.Request) {
		url := r.URL.String()
		if serversRE.MatchString(url) {
			rw.Write(serversJSON)
			rw.WriteHeader(200)
		} else if statusRE.MatchString(url) {
			rw.Write(statusJSON)
			rw.WriteHeader(200)
		} else {
			t.Errorf("Bad request: %#v", r)
			rw.WriteHeader(500)
		}
	}

	mainSrv := httptest.NewServer(http.HandlerFunc(h))
	//otherSrv := httptest.NewServer(http.HandlerFunc(h))

	rf := &ResolveFilter{}

	cl, err := NewClient(mainSrv.URL)
	if err != nil {
		t.Fatalf("Error building HTTP client: %#v", err)
	}
	poller := NewStatusPoller(cl, rf)

	var rState ResolveState
	go func() {
		var err error
		rState, err = poller.Start()
		if err != nil {
			t.Fatalf("Error starting poller: %#v", err)
		}
	}()

	select {
	case <-time.Tick(100 * time.Millisecond):
		t.Errorf("Happy path polling took more that 100ms")
	default:
		if rState != ResolveComplete {
			t.Errorf("Resolve state was %s not %s", rState, ResolveComplete)
		}
	}
}
