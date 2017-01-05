package sous

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
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
	Log.Vomit.SetOutput(os.Stderr)
	Log.Debug.SetOutput(os.Stderr)

	serversRE := regexp.MustCompile(`/servers$`)
	statusRE := regexp.MustCompile(`/status$`)
	var serversJSON, statusJSON []byte

	h := func(rw http.ResponseWriter, r *http.Request) {
		url := r.URL.String()
		if serversRE.MatchString(url) {
			rw.Write(serversJSON)
		} else if statusRE.MatchString(url) {
			rw.Write(statusJSON)
		} else {
			t.Errorf("Bad request: %#v", r)
			rw.WriteHeader(500)
			rw.Write([]byte{})
		}
	}

	mainSrv := httptest.NewServer(http.HandlerFunc(h))
	otherSrv := httptest.NewServer(http.HandlerFunc(h))

	repoName := "github.com/opentable/example"
	serversJSON = []byte(`{"servers":[{"clustername": "main", "url":"` + mainSrv.URL + `"},{"clustername": "other", "url":"` + otherSrv.URL + `"}]}`)
	statusJSON = []byte(`{
		"deployments": [
			{ "sourceid": {
				"location": { "repo": "` + repoName + `" }, "version": "1.0"
		} }
		], "completed": {"log":[{
			"deployid": { "manifestid": { "source": { "repo": "` + repoName + `" } },
			"desc": "unchanged"
		}}]}, "inprogress": {"log":[]}
  }`)

	rf := &ResolveFilter{
		Repo: repoName,
	}

	cl, err := NewClient(mainSrv.URL)
	if err != nil {
		t.Fatalf("Error building HTTP client: %#v", err)
	}
	poller := NewStatusPoller(cl, rf)

	testCh := make(chan ResolveState)
	go func() {
		rState, err := poller.Start()
		if err != nil {
			t.Fatalf("Error starting poller: %#v", err)
		}
		testCh <- rState
	}()

	timeout := 100 * time.Millisecond
	select {
	case <-time.Tick(timeout):
		t.Errorf("Happy path polling took more than %s", timeout)
	case rState := <-testCh:
		if rState != ResolveComplete {
			t.Errorf("Resolve state was %s not %s", rState, ResolveComplete)
		}
	}

	Log.Vomit.SetOutput(ioutil.Discard)
	Log.Debug.SetOutput(ioutil.Discard)
}
