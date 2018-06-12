package singularity

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var requestIDTests = []struct{ Repo, Dir, Flavor, Cluster, String string }{
	{
		"github.com/user/repo", "", "", "some-cluster",
		"repo---some_cluster",
		//"Sous_repo-some_cluster",
	},
	{
		"github.com/user/repo", "some/offset/dir", "", "some-cluster",
		"repo-some_offset_dir--some_cluster",
		//"Sous_repo_dir-some_cluster",
	},
	{
		"github.com/user/repo", "", "tasty-flavor", "some-cluster",
		"repo--tasty_flavor-some_cluster",
		//"Sous_repo-tasty_flavor-some_cluster",
	},
	{
		"github.com/user/repo", "some/offset/dir", "tasty-flavor", "some-cluster",
		"repo-some_offset_dir-tasty_flavor-some_cluster",
		//"Sous_repo_dir-tasty_flavor-some_cluster",
	},
}

func TestMakeRequestID(t *testing.T) {
	for _, test := range requestIDTests {
		input := sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: test.Repo,
					Dir:  test.Dir,
				},
				Flavor: test.Flavor,
			},
			Cluster: test.Cluster,
		}
		actual, err := MakeRequestID(input)
		if err != nil {
			t.Fatal(err)
		}
		expected := test.String

		if strings.Index(actual, expected) != 0 {
			t.Error(spew.Sprintf("%#v \n\tgot  %q; \n\twant %q", input, actual, expected))
		}
	}
}

func TestMakeRequestID_Long(t *testing.T) {
	actual, err := MakeRequestID(sous.DeploymentID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: "github.com/ihaveanincrediblylongname/andilikemyprojectstohaveincrediblylongnamestoo",
				Dir:  "and/also/i/bury/my/services/super/deep/in/the/build/tree/for/no/good/reason/blame/maven",
			},
			Flavor: "wellwehavetohaveaflavorforthisservicebecausethereseighteeninstancesofitwithweirdquirkstotheirconfig",
		},
		Cluster: "foo",
	})
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	// 99 is the maximum length of Singularity request IDs.
	if len(actual) > 99 {
		t.Errorf("Length of %q was %d which is longer than Singularity accepts by default.", actual, len(actual))
	}
}

func TestMakeRequestID_Collisions(t *testing.T) {
	tests := []struct{ repo, dir, flavor, cluster string }{
		{"github.com/user/repo", "", "", "some-cluster"},
		{"github.com/user/repo", "", "", "some_cluster"},
		{"github.com/user/re-po", "", "", "cluster"},
		{"github.com/user/re_po", "", "", "cluster"},
		{"github.com/user/re.po", "", "", "cluster"},
		{"github.com/user/repo", "some/offset/dir", "tasty-flavor", "some-cluster"},
		{"github.com/user/repo", "", "tasty_flavor", "some-cluster"},
		{"github.com/user/repo", "", "tasty-flavor", "some-cluster"},
		{"github.com/user/repo", "", "tasty.flavor", "some-cluster"},
		{"github.com/user/repo__some", "offset/dir", "tasty-flavor", "some-cluster"},
		{"github.com/user/repo", "some/offset/dir-tasty", "flavor", "some-cluster"},
		{"github.com/user/repo", "some/offset/dir", "tasty-flavor-some", "cluster"},
	}

	/*
			github_com_user_repo--some_cluster
			github_com_user_repo__some_offset_dir-tasty_flavor-some_cluster
		  github_com_user_repo__some__offset_dir-tasty_flavor-some_cluster
		  github_com_user_repo__some_offset_dir_tasty-flavor-some_cluster
		  github_com_user_repo__some_offset_dir-tasty_flavor_some-cluster
	*/

	for i, leftTest := range tests {
		left := sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: leftTest.repo,
					Dir:  leftTest.dir,
				},
				Flavor: leftTest.flavor,
			},
			Cluster: leftTest.cluster,
		}

		leftReqID, err := MakeRequestID(left)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(leftReqID)

		for j, rightTest := range tests {
			if i <= j {
				continue
			}

			right := sous.DeploymentID{
				ManifestID: sous.ManifestID{
					Source: sous.SourceLocation{
						Repo: rightTest.repo,
						Dir:  rightTest.dir,
					},
					Flavor: rightTest.flavor,
				},
				Cluster: rightTest.cluster,
			}

			rightReqID, err := MakeRequestID(right)
			if err != nil {
				t.Fatal(err)
			}
			if leftReqID == rightReqID {
				t.Error(spew.Sprintf("Collision! %q produced by \n\tboth %v \n\t and %v", leftReqID, left, right))
			}
		}
	}

}

func TestRectifyRecover(t *testing.T) {
	var err error
	expectedPrefix := "Panicked: What's that coming over the hill?!; stack trace:\n"
	ls, _ := logging.NewLogSinkSpy()
	func() {
		defer rectifyRecover("something", "TestRectifyRecover", &err, ls)
		panic("What's that coming over the hill?!")
	}()
	if err == nil {
		t.Fatalf("got nil, want error beginning %q", expectedPrefix)
	}
	actual := err.Error()
	if !strings.HasPrefix(actual, expectedPrefix) {
		t.Errorf("got error %q; want error with prefix %q", actual, expectedPrefix)
	}
}

// TestComputeDeployID tests a range of inputs from those which we expect to
// result in strings lower than the maximum length, up to strings that should
// result in truncation logic being invoked.
//
// Notably, it tests for off-by-one edge cases, by testing 16 and 17 character
// version strings which caused confusion in earlier implementations.
//
// It also tests for the 32/33 version string length boundary, at which we
// expect to begin truncating the version string itself.
func TestComputeDeployID(t *testing.T) {
	tests := []struct {
		VersionString, DeployIDPrefix string
		DeployIDLen                   int
	}{
		// Short version strings (below 17 characters) expect less than max deployId length.
		{"0.0.1", "0_0_1_", 38},
		{"0.0.2", "0_0_2_", 38},
		{"0.0.2-c", "0_0_2_", 40},

		// Exactly 15 charactes.
		{"0.0.2-789012345", "0_0_2_", 48},

		// Exactly 16 characters long, expect no truncation.
		{"0.0.2-7890123456", "0_0_2_", 49},
		{"10.12.5-90123456", "10_12_5_", 49},

		// Exactly 17 characters long, expect max deployId length.
		{"0.0.2-78901234567", "0_0_2_", 49},
		{"1.2.3-78901234567", "1_2_3_", 49},

		// Greater than 17 characters long, expect max deployId length.
		{"0.0.2-chr-eighteen", "0_0_2_", 49},
		{"0.0.2-thisversionissolongthatonewouldexpectittobetruncated", "0_0_2_", 49},
		{"10.12.5-thisversionissolongthatonewouldexpectittobetruncated", "10_12_5_", 49},

		// Exactly 32 chars long, expect full sanitised version string as prefix.
		{"10.12.5-32-chars-version-string", "10_12_5_32_chars_version_string_", 49},

		// Exactly 33 chars long, expect truncated sanitised version string as prefix.
		{"10.12.5-33-chars-version-stringX", "10_12_5_33_chars_version_string_", 49},
	}
	for _, test := range tests {
		inputVersion := test.VersionString
		expectedPrefix := test.DeployIDPrefix
		expectedLen := test.DeployIDLen
		input := &sous.Deployable{
			Deployment: &sous.Deployment{
				SourceID: sous.SourceID{
					Version: semv.MustParse(inputVersion),
				},
			},
		}
		actual := computeDeployID(input)
		if !strings.HasPrefix(actual, expectedPrefix) {
			t.Errorf("%s: got %q; want string prefixed %q", inputVersion, actual, expectedPrefix)
		}
		actualLen := len(actual)
		if actualLen != expectedLen {
			t.Errorf("%s: got length %d; want %d", inputVersion, actualLen, expectedLen)
		}
	}
}

func jsonRoundtrip(t *testing.T, start interface{}, end interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(start); err != nil {
		t.Fatalf("Couldn't serialize %v: %v", start, err)
	}

	dec := json.NewDecoder(buf)
	if err := dec.Decode(end); err != nil {
		t.Fatalf("Couldn't derialize %v: %v", buf.String(), err)
	}
}

// a zero Deployment doesn't quite work for us.
func baseDeployment() *sous.Deployment {
	startDep := sous.Deployment{} // trying with the zero...
	startDep.Cluster = &sous.Cluster{
		BaseURL: "http://dummy.cluster.example.com/",
	}
	startDep.Owners = sous.NewOwnerSet()
	startDep.Env = sous.Env{"A": "A"}
	startDep.Kind = sous.ManifestKindService
	startDep.DeployConfig.Resources = sous.Resources{"cpus": "0.1", "memory": "100", "ports": "1"}
	startDep.Startup = sous.Startup{SkipCheck: true}
	return &startDep
}

func matchedPair(t *testing.T, startDep *sous.Deployment) *sous.DeployablePair {
	reqID := "dummy-request"
	depID := "dummy-deploy"
	dockerName := "dummy-docker-image"
	// This happens in DiskStateManager on Read.
	flaws := startDep.Validate()
	require.Empty(t, flaws)

	deployable := sous.Deployable{
		Deployment: startDep,
		BuildArtifact: &sous.BuildArtifact{
			DigestReference: dockerName,
		},
	}

	ls, _ := logging.NewLogSinkSpy()

	_, aReq, err := singRequestFromDeployment(startDep, reqID, ls)
	assert.NoError(t, err)
	assert.NotNil(t, aReq)

	req := &dtos.SingularityRequest{}
	jsonRoundtrip(t, aReq, req)

	aDepReq, err := buildDeployRequest(deployable, reqID, depID, map[string]string{}, ls)
	assert.NoError(t, err)
	assert.NotNil(t, aDepReq)

	depReq := &dtos.SingularityDeployRequest{}
	jsonRoundtrip(t, aDepReq, depReq)

	log, _ := logging.NewLogSinkSpy()

	db := &deploymentBuilder{
		request: sRequest(req),
		deploy:  depReq.Deploy,
		log:     log,
	}

	assert.NoError(t, db.extractArtifactName(), "Could not extract ArtifactName (Docker image name) from SingularityDeploy.")
	assert.NoError(t, db.restoreFromMetadata(), "Could not determine cluster name based on SingularityDeploy Metadata.")
	assert.NoError(t, db.unpackDeployConfig(), "Could not convert data from a SingularityDeploy to a sous.Deployment.")
	assert.NoError(t, db.determineManifestKind(), "Could not determine SingularityRequestType.")
	assert.NoError(t, db.extractSchedule(), "Could not determine schedule.")

	post := &db.Target.Deployment

	return &sous.DeployablePair{
		Prior: &sous.Deployable{
			Deployment: startDep,
		},
		Post: &sous.Deployable{
			Deployment: post,
		},
	}
}

// XXX Not sure this is the right place for this test...
func TestStableDeployment(t *testing.T) {
	startDep := baseDeployment()
	pair := matchedPair(t, startDep)

	diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
	assert.False(t, diff)
	assert.Empty(t, diffs)

	assert.False(t, changesReq(pair), "Roundtrip of Deployment through Singularity DTOs reported as changing Request!")
	assert.False(t, changesDep(pair), "Roundtrip of Deployment through Singularity DTOs reported as changing Deploy!")
}

func TestScaling(t *testing.T) {
	startDep := baseDeployment()
	startDep.NumInstances = 5
	pair := matchedPair(t, startDep)
	pair.Prior.NumInstances = 16

	diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
	assert.True(t, diff)
	assert.NotEmpty(t, diffs)

	assert.True(t, changesReq(pair), "Updating number of instances reported as not changing Request!")
	assert.False(t, changesDep(pair), "Roundtrip of Deployment through Singularity DTOs reported as changing Deploy!")
}

func TestOwners(t *testing.T) {
	startDep := baseDeployment()
	startDep.Owners.Add("tom@opentable.com")
	pair := matchedPair(t, startDep)
	pair.Prior.Owners.Add("harry@opentable.com")

	diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
	assert.True(t, diff)
	assert.NotEmpty(t, diffs)

	assert.True(t, changesReq(pair), "Updating owners reported as not changing Request!")
	assert.False(t, changesDep(pair), "Roundtrip of Deployment through Singularity DTOs reported as changing Deploy!")
}

func TestScheduling(t *testing.T) {
	startDep := baseDeployment()
	startDep.Kind = sous.ManifestKindScheduled
	startDep.Schedule = "* 3 * * *"
	pair := matchedPair(t, startDep)

	assert.Equal(t, pair.Prior.Schedule, pair.Post.Schedule)
	assert.Equal(t, pair.Prior.Kind, pair.Post.Kind)

	pair.Prior.Schedule = "* 2 * * *"

	diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
	assert.True(t, diff)
	assert.NotEmpty(t, diffs)

	assert.True(t, changesReq(pair), "Updating schedule reported as not changing Request!")
	assert.False(t, changesDep(pair), "Roundtrip of Deployment through Singularity DTOs reported as changing Deploy!")
}

func TestSchedulingOnlyForScheduled(t *testing.T) {
	startDep := baseDeployment()
	startDep.Schedule = "* 3 * * *"
	pair := matchedPair(t, startDep)
	pair.Prior.Schedule = "* 2 * * *"

	diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
	assert.False(t, diff)
	assert.Empty(t, diffs)

	assert.False(t, changesReq(pair), "Changed schedule data for HTTP service treated as changing Request!")
	assert.False(t, changesDep(pair), "Changed schedule data for HTTP service treated as changing Deploy!")
}

func TestEnableStartupChangedDeployment(t *testing.T) {
	startDep := baseDeployment()
	startDep.Startup.SkipCheck = true

	pair := matchedPair(t, startDep)

	startDep.Startup.SkipCheck = false
	pair.Prior.Startup.CheckReadyProtocol = "HTTPS"

	diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
	assert.True(t, diff)
	assert.NotEmpty(t, diffs)

	assert.False(t, changesReq(pair), "Roundtrip of Deployment through Singularity DTOs reported as changing Request!")
	assert.True(t, changesDep(pair), "Startup checks change reported as not changing Deploy!")
}

func TestStartupChangedDeployment(t *testing.T) {
	startDep := baseDeployment()
	startDep.Startup.SkipCheck = false
	startDep.Startup.CheckReadyProtocol = "HTTP"
	pair := matchedPair(t, startDep)
	pair.Prior.Startup.CheckReadyProtocol = "HTTPS"

	diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
	assert.True(t, diff)
	assert.NotEmpty(t, diffs)

	assert.False(t, changesReq(pair), "Roundtrip of Deployment through Singularity DTOs reported as changing Request!")
	assert.True(t, changesDep(pair), "Startup checks change reported as not changing Deploy!")
}

func TestEnvChangedDeployment(t *testing.T) {
	startDep := baseDeployment()
	startDep.Env["TEST"] = "starting"
	pair := matchedPair(t, startDep)
	pair.Prior.Env["TEST"] = "ending"

	diff, diffs := pair.Prior.Deployment.Diff(pair.Post.Deployment)
	assert.True(t, diff)
	assert.NotEmpty(t, diffs)

	assert.False(t, changesReq(pair), "Roundtrip of Deployment through Singularity DTOs reported as changing Request!")
	assert.True(t, changesDep(pair), "Deployment environment change reported as not changing Deploy!")
}

func TestChangesReq(t *testing.T) {
	baseDep := sous.Deployment{}

	testPair := func(other *sous.Deployment) *sous.DeployablePair {
		return &sous.DeployablePair{
			Prior: &sous.Deployable{
				Deployment: &baseDep,
			},
			Post: &sous.Deployable{
				Deployment: other,
			},
		}
	}

	if changesReq(testPair(baseDep.Clone())) {
		t.Error("Unchanged deployment mis-reported to change requirement")
	}

	changed := baseDep.Clone()
	changed.DeployConfig.NumInstances = 100

	if !changesReq(testPair(changed)) {
		t.Error("Change in NumInstances ignored")
	}

	changed = baseDep.Clone()
	changed.Env["VAR"] = "VALUE"

	if changesReq(testPair(changed)) {
		t.Error("Non-request change (env var) to deployment mis-reported to change requirement")
	}
}

func TestChangesDep(t *testing.T) {
	baseDep := sous.Deployment{}

	testPair := func(other *sous.Deployment) *sous.DeployablePair {
		return &sous.DeployablePair{
			Prior: &sous.Deployable{
				Deployment: &baseDep,
			},
			Post: &sous.Deployable{
				Deployment: other,
			},
		}
	}

	if changesDep(testPair(baseDep.Clone())) {
		t.Error("Unchanged deployment mis-reported to changed deploy")
	}

	changed := baseDep.Clone()
	changed.DeployConfig.NumInstances = 100

	if changesDep(testPair(changed)) {
		t.Error("Change in NumInstances mis-reported as changed deploy")
	}

	pair := testPair(baseDep.Clone())
	pair.Post.Status = sous.DeployStatusFailed
	if !changesDep(pair) {
		t.Error("Failed post deploy not reported a changed.")
	}

	pair = testPair(baseDep.Clone())
	pair.Prior.Status = sous.DeployStatusFailed
	if !changesDep(pair) {
		t.Error("Failed prior deploy not reported a changed.")
	}

	changed = baseDep.Clone()
	changed.SourceID.Version.Minor = 12
	if !changesDep(testPair(changed)) {
		t.Error("Change to version on deployment reported as no change")
	}

	changed = baseDep.Clone()
	changed.Resources["cpus"] = "one million units!"
	if !changesDep(testPair(changed)) {
		t.Error("Change to cpus in resources on deployment reported as no change")
	}

	changed = baseDep.Clone()
	changed.Env["VAR"] = "VALUE"

	if !changesDep(testPair(changed)) {
		t.Error("Change to env var on deployment reported as no change")
	}

	changed = baseDep.Clone()
	changed.Volumes = append(changed.Volumes, &sous.Volume{})
	if !changesDep(testPair(changed)) {
		t.Error("Change to volumes on deployment reported as no change")
	}

	changed = baseDep.Clone()
	changed.Startup.CheckReadyURIPath = "/something/something/healthcheck"

	if !changesDep(testPair(changed)) {
		t.Error("Change to Startup on deployment reported as no change")
	}

	changed = baseDep.Clone()
	changed.Startup.SkipCheck = true
	baseDep.Startup.ConnectDelay = 100

	if !changesDep(testPair(changed)) {
		t.Errorf("Change to Startup on deployment reported as no change: \n%#v\n  vs\n%#v ", baseDep.Startup, changed.Startup)
	}
}

func TestPendingModification(t *testing.T) {
	drc := sous.NewDummyRectificationClient()
	deployer := NewDeployer(drc, logging.SilentLogSet())

	verStr := "0.0.1"
	dpl := &sous.Deployment{
		SourceID: sous.SourceID{
			Location: sous.SourceLocation{
				Repo: "fake.tld/org/project",
			},
			Version: semv.MustParse(verStr),
		},
		DeployConfig: sous.DeployConfig{
			NumInstances: 1,
			Resources:    sous.Resources{},
		},
		ClusterName: "cluster",
		Cluster: &sous.Cluster{
			BaseURL: "cluster",
		},
	}

	dp := &sous.DeployablePair{
		ExecutorData: &singularityTaskData{requestID: "reqid"},
		Post: &sous.Deployable{
			BuildArtifact: &sous.BuildArtifact{
				DigestReference: "build-artifact",
				Type:            "docker",
			},
			Deployment: dpl.Clone(),
			Status:     sous.DeployStatusPending,
		},
		Prior: &sous.Deployable{
			BuildArtifact: &sous.BuildArtifact{
				DigestReference: "build-artifact",
				Type:            "docker",
			},
			Deployment: dpl.Clone(),
			Status:     sous.DeployStatusActive,
		},
	}

	dpCh := make(chan *sous.DeployablePair)
	rezCh := make(chan sous.DiffResolution)

	go func() {
		for d := range dpCh {
			rezCh <- deployer.Rectify(d)
		}
	}()
	dpCh <- dp
	close(dpCh)

	rez := <-rezCh

	assert.Equal(t, sous.ModifyDiff, rez.Desc)
	assert.Zero(t, rez.Error)
	assert.Len(t, drc.Deployed, 0)
	assert.Len(t, drc.Created, 0)
	assert.Len(t, drc.Deleted, 0)
}

func TestModificationOfFailed(t *testing.T) {
	drc := sous.NewDummyRectificationClient()
	deployer := NewDeployer(drc, logging.SilentLogSet())

	verStr := "0.0.1"
	dpl := &sous.Deployment{
		SourceID: sous.SourceID{
			Location: sous.SourceLocation{
				Repo: "fake.tld/org/project",
			},
			Version: semv.MustParse(verStr),
		},
		DeployConfig: sous.DeployConfig{
			NumInstances: 1,
			Resources:    sous.Resources{},
		},
		ClusterName: "cluster",
		Cluster: &sous.Cluster{
			BaseURL: "cluster",
		},
	}

	dp := &sous.DeployablePair{
		ExecutorData: &singularityTaskData{requestID: "reqid"},
		Prior: &sous.Deployable{
			BuildArtifact: &sous.BuildArtifact{
				DigestReference: "build-artifact",
				Type:            "docker",
			},
			Deployment: dpl.Clone(),
			Status:     sous.DeployStatusActive,
		},
		Post: &sous.Deployable{
			BuildArtifact: &sous.BuildArtifact{
				DigestReference: "build-artifact",
				Type:            "docker",
			},
			Deployment: dpl.Clone(),
			Status:     sous.DeployStatusFailed,
		},
	}

	dpCh := make(chan *sous.DeployablePair)
	rezCh := make(chan sous.DiffResolution)

	go func() {
		for d := range dpCh {
			rezCh <- deployer.Rectify(d)
		}
	}()
	dpCh <- dp
	close(dpCh)

	rez := <-rezCh

	assert.Equal(t, sous.ModifyDiff, rez.Desc)
	assert.Error(t, rez.Error)
	assert.False(t, sous.IsTransientResolveError(rez.Error))
	assert.Len(t, drc.Deployed, 1)
	assert.Len(t, drc.Created, 0)
	assert.Len(t, drc.Deleted, 0)

}

func TestOptMaxHTTPReqsPerServer(t *testing.T) {
	d := NewDeployer(nil, logging.SilentLogSet()).(*deployer)
	if d.ReqsPerServer != DefaultMaxHTTPConcurrencyPerServer {
		t.Fatal("Not using default")
	}
	const x = 512
	if x == DefaultMaxHTTPConcurrencyPerServer {
		t.Fatal("Bad test, please specify a non-default value.")
	}
	deployer2 := NewDeployer(nil, logging.SilentLogSet(), OptMaxHTTPReqsPerServer(x)).(*deployer)
	if deployer2.ReqsPerServer != x {
		t.Fatalf("got %d; want %d", deployer2.ReqsPerServer, x)
	}
}
