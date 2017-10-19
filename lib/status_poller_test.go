package sous

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
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
	checkString(ResolveTasksStarting, "ResolveTasksStarting")
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
	testDir := ""

	rezErr := &ChangeError{}
	permErr := fmt.Errorf("something bad")

	deployment := func(version string, status DeployStatus) *Deployment {
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
				Desc: ResolutionType(desc),
			}
		}
		return &DiffResolution{
			Desc:  ResolutionType(desc),
			Error: &ErrorWrapper{MarshallableError: buildMarshableError(err)},
		}
	}

	testCompute := func(version string, intent *Deployment, current *DiffResolution, expected ResolveState) {
		sub := subPoller{
			idFilter: &ResolveFilter{
				Tag: NewResolveFieldMatcher(version),
			},
		}
		if actual, _ := sub.computeState(intent, current); expected != actual {
			t.Errorf("sub.computeState(%v, %v) -> %v != %v", intent, current, actual, expected)
		}
	}

	testCompute("1.0", nil, nil, ResolveNotStarted)
	testCompute("1.0", deployment("0.9", DeployStatusAny), nil, ResolveNotVersion)
	testCompute("1.0", deployment("1.0", DeployStatusAny), nil, ResolvePendingRequest)

	testCompute("1.0", deployment("1.0", DeployStatusAny), diffRez("update", nil), ResolveInProgress) //known update , no outcome yet

	testCompute("1.0", deployment("1.0", DeployStatusAny), diffRez("unchanged", rezErr), ResolveErredRez)
	testCompute("1.0", deployment("1.0", DeployStatusAny), diffRez("unchanged", permErr), ResolveFailed)

	testCompute("1.0", deployment("1.0", DeployStatusAny), diffRez("unchanged", nil), ResolveComplete)

	testCompute("1.0", deployment("1.0", DeployStatusPending), diffRez("coming", nil), ResolveTasksStarting)
}

func TestStatusPoller_updateState(t *testing.T) {
	assert := assert.New(t)

	sp := &StatusPoller{
		statePerCluster: map[string]*pollerState{
			"one": &pollerState{
				LastResult: pollResult{stat: ResolveNotPolled},
				//LastCycle:  true,
			},
			"two": &pollerState{
				LastResult: pollResult{stat: ResolveNotPolled},
				//LastCycle:  true,
			},
		},
		status: ResolveNotPolled,
	}

	// keep track of ordered results so far for better test output.
	var resultsSoFar []pollResult
	resultsSoFarStr := func() string {
		buf := &bytes.Buffer{}
		for _, r := range resultsSoFar {
			fmt.Fprintf(buf, "cluster: %s; state: %s; ResolveID: %s\n",
				r.url, r.stat, r.resolveID)
		}
		return buf.String()
	}

	assertStatus := func(expected ResolveState) {
		actual := sp.status
		if actual != expected {
			t.Errorf("got %s; want %s (after %d results):\n%s",
				actual, expected, len(resultsSoFar), resultsSoFarStr())
		}
	}
	result := func(clusterName, resolveID string, status ResolveState) {
		result := pollResult{
			url:       clusterName,
			stat:      status,
			resolveID: resolveID,
		}
		sp.nextSubStatus(result)
		resultsSoFar = append(resultsSoFar, result)
		sp.updateStatus()
	}

	first := "2017-10-18T14:29:37.115976034Z"
	second := "2018-11-18T14:29:37.115976034Z"

	/// TODO: tests for "competing states"

	assertStatus(ResolveNotPolled)

	assert.False(sp.finished(), "StatusPoller reported finished: %s", sp.status)

	result("one", first, ResolveNotPolled)
	result("two", first, ResolveNotPolled)
	assertStatus(ResolveNotPolled)

	// One moves to ResolveNotStarted, overall now ResolveNotStarted.
	result("one", first, ResolveNotStarted)
	assertStatus(ResolveNotStarted)
	assert.False(sp.finished(), "StatusPoller reported finished: %s", sp.status)

	// Two also moved to ResolveNotStarted, overall still ResolveNotStarted.
	result("two", first, ResolveNotStarted)
	assertStatus(ResolveNotStarted)

	// One moved to ResolveNotVersion, overall ResolveNotVersion
	result("one", first, ResolveNotVersion)
	assertStatus(ResolveNotVersion)

	// One moved to ResolveInProgress, overall ResolveInProgress
	result("one", first, ResolveInProgress)
	assertStatus(ResolveInProgress)

	// One moved to ResolveTasksStarting, overall ResolveInProgress
	// because still on first resolveID.
	result("one", first, ResolveTasksStarting)
	assertStatus(ResolveInProgress)

	// Both move to ResolveComplete in first cycle, overall ResolveComplete.
	result("one", first, ResolveComplete)
	result("two", first, ResolveComplete)
	assertStatus(ResolveComplete)

	// One moves to ResolveFailed in first cycle, overall ResolveInProgress
	result("one", first, ResolveFailed)
	assertStatus(ResolveInProgress)

	// Two moves to ResolveNotStarted (second resolveID).
	// Overall still ResolveInProgress because two still on first resolveID.
	result("two", second, ResolveNotStarted)
	assertStatus(ResolveInProgress)

	// One and two both move to ResolveNotPolled (second resolveID).
	// Overall still ResolveInProgress because that's the highest
	// so far.
	result("one", second, ResolveNotPolled)
	result("two", second, ResolveNotPolled)
	assertStatus(ResolveInProgress)

	// One moves to ResolveFailed in last cycle. Overall failed.
	result("one", second, ResolveFailed)
	assertStatus(ResolveFailed)

	// One moves to ResolveComplete in last cycle, still in progress.
	result("one", second, ResolveComplete)
	assertStatus(ResolveInProgress)

	result("two", second, ResolveFailed)
	assertStatus(ResolveFailed)

	result("two", second, ResolveComplete)
	assertStatus(ResolveComplete)
}

func TestStatusPoller(t *testing.T) {
	serversRE := regexp.MustCompile(`/servers$`)
	statusRE := regexp.MustCompile(`/status$`)
	gdmRE := regexp.MustCompile(`/gdm$`)
	var gdmJSON, serversJSON, statusJSON, statusJSON2 []byte

	statusCalled := false
	handleMutex := sync.Mutex{}

	h := func(rw http.ResponseWriter, r *http.Request) {
		// For testing purposes, we want to ensure we handle
		// responses one at a time since statusCalled must
		// be false on the first call and true on the second.
		// The race detector picked up this issue.
		handleMutex.Lock()
		defer handleMutex.Unlock()
		url := r.URL.String()
		if serversRE.MatchString(url) {
			rw.Write(serversJSON)
		} else if statusRE.MatchString(url) {
			if !statusCalled {
				statusCalled = true
				rw.Write(statusJSON)
			} else {
				rw.Write(statusJSON2)
			}
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
				},
				"flavor": "canhaz"
			},
			{
				"clustername": "main",
				"sourceid": {
					"location": "` + repoName + `",
					"version": "1.0.1+1234"
				},
				"flavor": "canhaz"
			}
		]
	}`)
	statusJSON = []byte(`{
		"deployments": [
			{
				"sourceid": {
					"location": "` + repoName + `",
					"version": "1.0.1+1234"
				},
				"flavor": "canhaz"
			}
		],
		"completed": {
			"intended": [ {
				"sourceid": {
					"location": "` + repoName + `",
					"version": "1.0.1+1234"
				},
				"flavor": "canhaz"
			} ],
			"log":[ {
					"manifestid": "` + repoName + `~canhaz",
					"desc": "unchanged"
				} ]
		},
		"inprogress": {"log":[], "started": "2017-10-11T14:26:05.975369893Z"}
	}`)
	statusJSON2 = []byte(`{
		"deployments": [
			{
				"sourceid": {
					"location": "` + repoName + `",
					"version": "1.0.1+1234"
				},
				"flavor": "canhaz"
			}
		],
		"completed": {
			"intended": [ {
				"sourceid": {
					"location": "` + repoName + `",
					"version": "1.0.1+1234"
				},
				"flavor": "canhaz"
			} ],
			"log":[ {
					"manifestid": "` + repoName + `~canhaz",
					"desc": "unchanged"
				} ]
		},
		"inprogress": {"log":[], "started": "2018-10-11T14:27:05.975369893Z"}
	}`)

	rf := &ResolveFilter{
		Repo: NewResolveFieldMatcher(repoName),
	}
	rf.SetTag("")
	// XXX Flavor
	//   and deploy should probably not treat Flavor as * by default (instead "")

	cl, err := restful.NewClient(mainSrv.URL, logging.SilentLogSet())
	if err != nil {
		t.Fatalf("Error building HTTP client: %#v", err)
	}
	poller := NewStatusPoller(cl, rf, User{Name: "Test User"}, logging.SilentLogSet())

	testCh := make(chan ResolveState)
	go func() {
		rState, err := poller.Wait(context.Background())
		if err != nil {
			t.Errorf("Error starting poller: %#v", err)
		}
		testCh <- rState
	}()

	timeout := 3000 * time.Millisecond
	select {
	case <-time.After(timeout):
		t.Errorf("Happy path polling took more than %s", timeout)
	case rState := <-testCh:
		if rState != ResolveComplete {
			t.Errorf("Resolve state was %s not %s", rState, ResolveComplete)
		}
	}
}

func TestStatusPoller_OldServer2(t *testing.T) {
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
		Repo: NewResolveFieldMatcher(repoName),
	}
	rf.SetTag("")

	cl, err := restful.NewClient(mainSrv.URL, logging.SilentLogSet())
	if err != nil {
		t.Fatalf("Error building HTTP client: %#v", err)
	}
	poller := NewStatusPoller(cl, rf, User{Name: "Test User"}, logging.SilentLogSet())

	testCh := make(chan ResolveState)
	go func() {
		rState, err := poller.Wait(context.Background())
		if err != nil {
			t.Errorf("Error starting poller: %#v", err)
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

func TestStatusPoller_MesosFailed(t *testing.T) {
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
			"intended": [ {
					"sourceid": {
						"location": "` + repoName + `",
						"version": "1.0.1+1234"
					}
				} ],
			"log":[ {
					"manifestid": "` + repoName + `",
					"desc": "unchanged",
					"error": {
					  "type": "FailedStatusError",
						"string": "Deploy failed on Singularity."
					}
				} ]
		},
		"inprogress": {"log":[]}
	}`)

	rf := &ResolveFilter{
		Repo: NewResolveFieldMatcher(repoName),
	}
	rf.SetTag("")

	cl, err := restful.NewClient(mainSrv.URL, logging.SilentLogSet())
	if err != nil {
		t.Fatalf("Error building HTTP client: %#v", err)
	}
	poller := NewStatusPoller(cl, rf, User{Name: "Test User"}, logging.SilentLogSet())

	testCh := make(chan ResolveState)
	go func() {
		rState, err := poller.Wait(context.Background())
		if err != nil {
			t.Errorf("Error starting poller: %#v", err)
		}
		t.Logf("Returned state: %#v", rState)
		testCh <- rState
	}()

	timeout := 300 * time.Millisecond
	select {
	case <-time.After(timeout):
		t.Errorf("Happy path polling took more than %s", timeout)
	case rState := <-testCh:
		if rState != ResolveFailed {
			t.Errorf("Resolve state was %s not %s", rState, ResolveFailed)
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
		Repo: NewResolveFieldMatcher(repoName),
	}

	cl, err := restful.NewClient(mainSrv.URL, logging.SilentLogSet())
	if err != nil {
		t.Fatalf("Error building HTTP client: %#v", err)
	}
	poller := NewStatusPoller(cl, rf, User{Name: "Test User"}, logging.SilentLogSet())

	testCh := make(chan ResolveState)
	go func() {
		rState, err := poller.Wait(context.Background())
		if err != nil {
			t.Errorf("Error starting poller: %#v", err)
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
		Repo: NewResolveFieldMatcher("github.com/something/summat"),
	}

	cl, err := restful.NewClient(mainSrv.URL, logging.SilentLogSet())
	if err != nil {
		t.Fatalf("Error building HTTP client: %#v", err)
	}
	poller := NewStatusPoller(cl, rf, User{Name: "Test User"}, logging.SilentLogSet())

	testCh := make(chan ResolveState)
	go func() {
		rState, err := poller.Wait(context.Background())
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
