package singularity

import (
	"reflect"
	"strings"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

var requestIDTests = []struct {
	DeployID sous.DeploymentID
	String   string
}{
	// repo, cluster
	{
		DeployID: sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user/repo",
				},
			},
			Cluster: "some-cluster",
		},
		String: "github.com>user>repo::some-cluster",
	},
	// repo, dir, cluster
	{
		DeployID: sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user/repo",
					Dir:  "some/offset/dir",
				},
			},
			Cluster: "some-cluster",
		},
		String: "github.com>user>repo,some>offset>dir::some-cluster",
	},
	// repo, flavor, cluster
	{
		DeployID: sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user/repo",
				},
				Flavor: "tasty-flavor",
			},
			Cluster: "some-cluster",
		},
		String: "github.com>user>repo:tasty-flavor:some-cluster",
	},
	// repo, dir, flavor, cluster
	{
		DeployID: sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user/repo",
					Dir:  "some/offset/dir",
				},
				Flavor: "tasty-flavor",
			},
			Cluster: "some-cluster",
		},
		String: "github.com>user>repo,some>offset>dir:tasty-flavor:some-cluster",
	},
}

func TestMakeRequestID(t *testing.T) {
	for _, test := range requestIDTests {
		input := test.DeployID
		expected := test.String
		actual := MakeRequestID(input)
		if actual != expected {
			t.Errorf("%#v got %q; want %q", input, actual, expected)
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
					Repo: "reqid",
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
					Repo: "reqid",
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
	if len(deployID) > maxDeployIDLen {
		t.Fatalf(logLenTmpl, idLen, maxDeployIDLen)
	} else {
		t.Logf(logLenTmpl, idLen, maxDeployIDLen)
	}
}

func TestPendingModification(t *testing.T) {
	drc := sous.NewDummyRectificationClient()
	deployer := NewDeployer(drc)

	verStr := "0.0.1"
	dpl := &sous.Deployment{
		SourceID: sous.SourceID{
			Location: sous.SourceLocation{
				Repo: "reqid",
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

	assert.Equal(t, rez.Desc, sous.ModifyDiff)
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
				Repo: "reqid",
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
