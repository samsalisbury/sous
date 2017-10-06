package sous

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
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
		pollChans: map[string]ResolveState{
			"one": ResolveInProgress,
			"two": ResolveErredHTTP,
		},
		status: ResolveNotStarted,
	}

	assertStatus := func(status ResolveState) {
		assert.Equal(sp.status, status, "StatusPoller total state was %s, expected %s", sp.status, status)
	}

	/// TODO: tests for "competing states"

	assert.False(sp.finished(), "StatusPoller reported finished: %s", sp.status)
	assertStatus(ResolveInProgress)

	sp.pollChans["one"] = ResolveTasksStarting

	assert.False(sp.finished(), "StatusPoller reported finished: %s", sp.status)
	assertStatus(ResolveTasksStarting)

	sp.pollChans["one"] = ResolveComplete
	sp.pollChans["two"] = ResolveComplete

	assert.True(sp.finished(), "StatusPoller reported NOT finished: %s", sp.status)
	assertStatus(ResolveComplete)

	sp.pollChans["one"] = ResolveComplete
	sp.pollChans["two"] = ResolveFailed

	assert.True(sp.finished(), "StatusPoller reported NOT finished: %s", sp.status)
	assertStatus(ResolveFailed)
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
		"inprogress": {"log":[]}
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
