package singularity

import (
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

// TestComputeDeployID tests a range of inputs from those which we expect to
// result in strings lower than the maximum length, up to strings that should
// result in truncation logic being invoked.
//
// Notably, it tests for off-by-one edge cases, by testing 16 and 17 character
// version strings which caused confusion in earlier implementations.
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
		{"10.12.5-thisversionissolongthatonewouldexpectittobetruncated", "10_12_5_", 49}}
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
