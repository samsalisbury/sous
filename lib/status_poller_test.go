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

func TestResolveState_String(t *testing.T) {
	checkString := func(r ResolveState, expected string) {
		actual := r.String()
		if actual != expected {
			t.Errorf("ResolveState %[1]d (%[1]s) String() => %q, should be %q", r, actual, expected)
		}
	}

	checkString(ResolveNotPolled, "ResolveNotPolled")
	checkString(ResolveNotStarted, "ResolveNotStarted")
	checkString(ResolveNotVersion, "ResolveNotVersion")
	checkString(ResolvePendingRequest, "ResolvePendingRequest")
	checkString(ResolveInProgress, "ResolveInProgress")
	checkString(ResolveErredHTTP, "ResolveErredHTTP")
	checkString(ResolveErredRez, "ResolveErredRez")
	checkString(ResolveComplete, "ResolveComplete")
	checkString(ResolveState(1e6), "unknown (oops)")

	for rs := ResolveNotStarted; rs <= ResolveMAX; rs++ {
		if rs.String() == "unknown (oops)" {
			t.Errorf("ResolveState %d doesn't have a string", rs)
		}
	}
}

func TestSubPoller_ComputeState(t *testing.T) {
	testRepo := "github.com/opentable/example"
	testDir := "test"

	rezErr := &ChangeError{}
	permErr := fmt.Errorf("something bad")

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
		if err == nil {
			return &DiffResolution{
				Desc: desc,
			}
		}
		return &DiffResolution{
			Desc:  desc,
			Error: &ErrorWrapper{MarshallableError: buildMarshableError(err)},
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
	testCompute("1.0", versionDep("1.0"), diffRez("create", permErr), diffRez("update", nil), ResolveInProgress)

	testCompute("1.0", versionDep("1.0"), diffRez("unchanged", rezErr), nil, ResolveErredRez)
	testCompute("1.0", versionDep("1.0"), nil, diffRez("unchanged", rezErr), ResolveErredRez)
	testCompute("1.0", versionDep("1.0"), diffRez("unchanged", nil), diffRez("unchanged", rezErr), ResolveErredRez)
	testCompute("1.0", versionDep("1.0"), diffRez("create", rezErr), diffRez("unchanged", rezErr), ResolveErredRez)
	testCompute("1.0", versionDep("1.0"), diffRez("unchanged", nil), diffRez("unchanged", permErr), ResolveFailed)
	testCompute("1.0", versionDep("1.0"), diffRez("create", rezErr), diffRez("unchanged", permErr), ResolveFailed)

	testCompute("1.0", versionDep("1.0"), diffRez("unchanged", nil), nil, ResolveComplete)
	testCompute("1.0", versionDep("1.0"), nil, diffRez("unchanged", nil), ResolveComplete)
	testCompute("1.0", versionDep("1.0"), diffRez("create", rezErr), diffRez("unchanged", nil), ResolveComplete)
	testCompute("1.0", versionDep("1.0"), diffRez("create", permErr), diffRez("unchanged", nil), ResolveComplete)
}

func TestStatusPoller(t *testing.T) {
	serversRE := regexp.MustCompile(`/servers$`)
	statusRE := regexp.MustCompile(`/status$`)
	gdmRE := regexp.MustCompile(`/gdm$`)
	var gdmJSON, serversJSON, statusJSON []byte

	h := func(rw http.ResponseWriter, r *http.Request) {
		url := r.URL.String()
		if serversRE.MatchString(url) {
			rw.Write(serversJSON)
		} else if statusRE.MatchString(url) {
			rw.Write(statusJSON)
		} else if gdmRE.MatchString(url) {
			rw.Write(gdmJSON)
		} else {
			t.Errorf("Bad request: %#v", r)
			rw.WriteHeader(500)
			rw.Write([]byte{})
		}
	}

	mainSrv := httptest.NewServer(http.HandlerFunc(h))
	otherSrv := httptest.NewServer(http.HandlerFunc(h))

	repoName := "github.com/opentable/example"

	serversJSON = []byte(`{
		"servers": [
			{"clustername": "main", "url":"` + mainSrv.URL + `"},
			{"clustername": "other", "url":"` + otherSrv.URL + `"}
		]
	}`)
	gdmJSON = []byte(`{
		"deployments": [
			{
				"clustername": "other",
				"sourceid": {
					"location": "` + repoName + `",
					"version": "1.0.1+1234"
				}
			},
			{
				"clustername": "main",
				"sourceid": {
					"location": "` + repoName + `",
					"version": "1.0.1+1234"
				}
			}
		]
	}`)
	statusJSON = []byte(`{
		"deployments": [
			{
				"sourceid": {
					"location": "` + repoName + `",
					"version": "1.0.1+1234"
				}
			}
		],
		"completed": {
			"log":[ {
					"manifestid": "` + repoName + `",
					"desc": "unchanged"
				} ]
		},
		"inprogress": {"log":[]}
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
	case <-time.After(timeout):
		t.Errorf("Happy path polling took more than %s", timeout)
	case rState := <-testCh:
		if rState != ResolveComplete {
			t.Errorf("Resolve state was %s not %s", rState, ResolveComplete)
		}
	}
}

func TestStatusPoller_NotIntended(t *testing.T) {
	serversRE := regexp.MustCompile(`/servers$`)
	statusRE := regexp.MustCompile(`/status$`)
	gdmRE := regexp.MustCompile(`/gdm$`)
	var gdmJSON, serversJSON, statusJSON []byte

	h := func(rw http.ResponseWriter, r *http.Request) {
		url := r.URL.String()
		if serversRE.MatchString(url) {
			rw.Write(serversJSON)
		} else if statusRE.MatchString(url) {
			rw.Write(statusJSON)
		} else if gdmRE.MatchString(url) {
			rw.Write(gdmJSON)
		} else {
			t.Errorf("Bad request: %#v", r)
			rw.WriteHeader(500)
			rw.Write([]byte{})
		}
	}

	mainSrv := httptest.NewServer(http.HandlerFunc(h))
	otherSrv := httptest.NewServer(http.HandlerFunc(h))

	repoName := "github.com/opentable/example"

	serversJSON = []byte(`{
		"servers": [
			{"clustername": "main", "url":"` + mainSrv.URL + `"},
			{"clustername": "other", "url":"` + otherSrv.URL + `"}
		]
	}`)
	gdmJSON = []byte(`{
		"deployments": [ ]
	}`)
	statusJSON = []byte(`{
		"deployments": [
			{
				"sourceid": {
					"location": "` + repoName + `",
					"version": "1.0.1+1234"
				}
			}
		],
		"completed": {
			"log":[ {
					"manifestid": "` + repoName + `",
					"desc": "unchanged"
				} ]
		},
		"inprogress": {"log":[]}
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
	case <-time.After(timeout):
		t.Errorf("Empty subpoller polling took more than %s", timeout)
	case rState := <-testCh:
		if rState != ResolveNotIntended {
			t.Errorf("Resolve state was %s not %s", rState, ResolveNotIntended)
		}
	}
}

func TestStatusPoller_OldServer(t *testing.T) {
	serversRE := regexp.MustCompile(`/servers$`)
	statusRE := regexp.MustCompile(`/status$`)

	h := func(rw http.ResponseWriter, r *http.Request) {
		url := r.URL.String()
		if serversRE.MatchString(url) {
			rw.WriteHeader(404)
			rw.Write([]byte{})
		} else if statusRE.MatchString(url) {
			rw.WriteHeader(404)
			rw.Write([]byte{})
		} else {
			t.Errorf("Bad request: %#v", r)
			rw.WriteHeader(500)
			rw.Write([]byte{})
		}
	}

	mainSrv := httptest.NewServer(http.HandlerFunc(h))

	rf := &ResolveFilter{
		Repo: "github.com/something/summat",
	}

	cl, err := NewClient(mainSrv.URL)
	if err != nil {
		t.Fatalf("Error building HTTP client: %#v", err)
	}
	poller := NewStatusPoller(cl, rf)

	testCh := make(chan ResolveState)
	go func() {
		rState, err := poller.Start()
		if err == nil {
			t.Errorf("No error starting poller: %#v", err)
		}
		testCh <- rState
	}()

	timeout := 100 * time.Millisecond
	select {
	case <-time.After(timeout):
		t.Errorf("Sad path polling took more than %s", timeout)
	case rState := <-testCh:
		if rState != ResolveFailed {
			t.Errorf("Resolve state was %s not %s", rState, ResolveFailed)
		}
	}
}
