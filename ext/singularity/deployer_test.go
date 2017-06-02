package singularity

import (
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	sous "github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
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
	if len(actual) >= 100 {
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
	expected := "Panicked"
	func() {
		defer rectifyRecover("something", "TestRectifyRecover", &err)
		panic("What's that coming over the hill?!")
	}()
	if err == nil {
		t.Fatalf("got nil, want error %q", expected)
	}
	actual := err.Error()
	if actual != expected {
		t.Errorf("got error %q; want %q", actual, expected)
	}
}

func TestShortComputeDeployID(t *testing.T) {
	verStr := "0.0.1"
	logTmpl := "Provided version string:%s DeployID:%#v"
	d := &sous.Deployable{
		BuildArtifact: &sous.BuildArtifact{
			Name: "build-artifact",
			Type: "docker",
		},
		Deployment: &sous.Deployment{
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
		},
	}

	deployID := computeDeployID(d)
	parsedDeployID := strings.Split(deployID, "_")[0:3]
	if reflect.DeepEqual(parsedDeployID, strings.Split(verStr, ".")) {
		t.Logf(logTmpl, verStr, deployID)
	} else {
		t.Fatalf(logTmpl, verStr, deployID)
	}
	t.Logf("LENGTH: %d", len(deployID))
}

func TestLongComputeDeployID(t *testing.T) {
	verStr := "0.0.2-thisversionissolongthatonewouldexpectittobetruncated"
	logTmpl := "Provided version string:%s DeployID:%#v"
	d := &sous.Deployable{
		BuildArtifact: &sous.BuildArtifact{
			Name: "build-artifact",
			Type: "docker",
		},
		Deployment: &sous.Deployment{
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
		},
	}

	deployID := computeDeployID(d)
	parsedDeployID := strings.Split(deployID, "_")[0:3]
	if reflect.DeepEqual(parsedDeployID, strings.Split("0.0.2", ".")) {
		t.Logf(logTmpl, verStr, deployID)
	} else {
		t.Fatalf(logTmpl, verStr, deployID)
	}

	idLen := len(deployID)
	logLenTmpl := "Got length:%d Max length:%d"
	if len(deployID) >= 50 { // 50 is how our Singularity is configured
		t.Fatalf(logLenTmpl, idLen, maxDeployIDLen)
	} else {
		t.Logf(logLenTmpl, idLen, maxDeployIDLen)
	}
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
	hcpath := "/something/something/healthcheck"
	changed.Startup.CheckReadyURIPath = &hcpath

	if !changesDep(testPair(changed)) {
		t.Error("Change to Startup on deployment reported as no change")
	}
}

func TestPendingModification(t *testing.T) {
	drc := sous.NewDummyRectificationClient()
	deployer := NewDeployer(drc)

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
				Name: "build-artifact",
				Type: "docker",
			},
			Deployment: dpl.Clone(),
			Status:     sous.DeployStatusPending,
		},
		Prior: &sous.Deployable{
			BuildArtifact: &sous.BuildArtifact{
				Name: "build-artifact",
				Type: "docker",
			},
			Deployment: dpl.Clone(),
			Status:     sous.DeployStatusActive,
		},
	}

	dpCh := make(chan *sous.DeployablePair)
	rezCh := make(chan sous.DiffResolution)

	go deployer.RectifyModifies(dpCh, rezCh)
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
	deployer := NewDeployer(drc)

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
				Name: "build-artifact",
				Type: "docker",
			},
			Deployment: dpl.Clone(),
			Status:     sous.DeployStatusFailed,
		},
		Prior: &sous.Deployable{
			BuildArtifact: &sous.BuildArtifact{
				Name: "build-artifact",
				Type: "docker",
			},
			Deployment: dpl.Clone(),
			Status:     sous.DeployStatusActive,
		},
	}

	dpCh := make(chan *sous.DeployablePair)
	rezCh := make(chan sous.DiffResolution)

	go deployer.RectifyModifies(dpCh, rezCh)
	dpCh <- dp
	close(dpCh)

	rez := <-rezCh

	assert.Equal(t, rez.Desc, sous.ModifyDiff)
	assert.Error(t, rez.Error)
	assert.False(t, sous.IsTransientResolveError(rez.Error))
	assert.Len(t, drc.Deployed, 1)
	assert.Len(t, drc.Created, 0)
	assert.Len(t, drc.Deleted, 0)

}
