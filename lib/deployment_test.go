package sous

import (
	"reflect"
	"testing"

	"github.com/opentable/sous/util/allfields"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentDiffAnalysis(t *testing.T) {
	exemptions := []string{
		// we depend on ClusterName to uniquely identify clusters,
		// so we don't compare the cluster itself
		"Deployment.Cluster",
		"Deployment.Cluster.Name",
		"Deployment.Cluster.Kind",
		"Deployment.Cluster.BaseURL",
		"Deployment.Cluster.Env",
		"Deployment.Cluster.AllowedAdvisories",
		"Deployment.Cluster.Startup",
		"Deployment.Cluster.Startup.SkipCheck",
		"Deployment.Cluster.Startup.CheckReadyURIPath",
		"Deployment.Cluster.Startup.CheckReadyURITimeout",
		"Deployment.Cluster.Startup.Timeout",
		"Deployment.Cluster.Startup.ConnectInterval",
		"Deployment.Cluster.Startup.CheckReadyFailureStatuses",
		"Deployment.Cluster.Startup.CheckReadyRetries",
		"Deployment.Cluster.Startup.CheckReadyProtocol",
		"Deployment.Cluster.Startup.CheckReadyInterval",
		"Deployment.Cluster.Startup.ConnectDelay",
		"Deployment.Cluster.Startup.CheckReadyPortIndex",
		// SourceID.Location is incorporated into the value of ID(),
		// is is compared directly - Repo and Dir are compared implicitly thereby
		"Deployment.SourceID.Location.Repo",
		"Deployment.SourceID.Location.Dir",
		"Deployment.User",
		"Deployment.User.Name",
		"Deployment.User.Email",
		/*
			"Deployment.Owners",
			"Deployment.DeployConfig.Args",
			"Deployment.Args",
		*/
	}

	ast := allfields.ParseDir(".")
	tree := allfields.ExtractTree(ast, "Deployment")
	untouched := allfields.ConfirmTree(tree, ast, "Diff").Exempt(exemptions)

	assert.Empty(t, untouched)
}

func TestDeploymentClone(t *testing.T) {
	vers := semv.MustParse("1.2.3-test+thing")
	vols := Volumes{
		{"h", "c", "RO"},
		{"h2", "c2", "RW"},
	}
	original := &Deployment{
		DeployConfig: DeployConfig{
			Resources:    Resources{},
			Env:          Env{},
			NumInstances: 3,
			Volumes:      vols,
		},
		SourceID: SourceID{
			Location: SourceLocation{
				Repo: "one",
				Dir:  "two",
			},
			Version: vers,
		},
	}

	cloned := original.Clone()
	assert.Len(t, cloned.Volumes, 2)
	assert.Equal(t, cloned.Volumes[0].Container, "c")
	assert.True(t, reflect.DeepEqual(original, cloned))

	original.Volumes = Volumes{}

	assert.Len(t, cloned.Volumes, 2)
}

func TestCanonName(t *testing.T) {
	assert := assert.New(t)

	vers, _ := semv.Parse("1.2.3-test+thing")
	dep := Deployment{
		SourceID: SourceID{
			Location: SourceLocation{
				Repo: "one",
				Dir:  "two",
			},
			Version: vers,
		},
	}
	str := dep.SourceID.Location.String()
	assert.Regexp("one", str)
	assert.Regexp("two", str)
}

func TestBuildDeployment(t *testing.T) {
	assert := assert.New(t)
	m := &Manifest{
		Source: SourceLocation{},
		Owners: []string{"test@testerson.com"},
		Kind:   ManifestKindService,
	}
	sp := DeploySpec{
		DeployConfig: DeployConfig{
			Resources:    Resources{},
			Env:          Env{},
			NumInstances: 3,
			Volumes: Volumes{
				&Volume{"h", "c", "RO"},
			},
		},
		Version:     semv.MustParse("1.2.3"),
		clusterName: "cluster.name",
	}
	var ih []DeploySpec
	nick := "cn"

	cluster := &Cluster{BaseURL: "http://not"}

	d, err := BuildDeployment(m, nick, cluster, sp, ih)

	if assert.NoError(err) {
		if assert.Len(d.DeployConfig.Volumes, 1) {
			assert.Equal("c", d.DeployConfig.Volumes[0].Container)
		}
		assert.Equal(nick, d.ClusterName)
	}
}

func TestDeployment_String(t *testing.T) {
	// The key (name) of each test case looks roughly like what you would expect
	// from fmt.Sprintf("%+v"). We do not use this since we are testing
	// representations of the test case, so there is a possibility that doing
	// this would beg the same question as the test and thus make output
	// confusing.
	testCases := map[string]struct {
		in   *Deployment
		want string
	}{
		// The nil test case is what inspired these tests as we had a panic
		// during usage of 'sous deploy' because of it.
		"nil": {
			in:   nil,
			want: "<nil>",
		},
		"&Deployment{}": {
			in: &Deployment{},
			// Current value of want reflects current reality.
			// I think we can do better than this representation...
			want: ",0.0.0 \"\" @ <unknown> #0 {false 0 0 0   0 <nil> 0 0 0} map[] : map[] []",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var recovered interface{}
			var got string
			func() {
				defer func() { recovered = recover() }()
				got = tc.in.String()
			}()
			if recovered != nil {
				t.Errorf("panicked with %v", recovered)
			}
			if got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}
