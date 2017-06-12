package sous

import (
	"fmt"
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

		// SourceID.Location is incorporated into the value of ID(),
		// is is compared directly - Repo and Dir are compared implicitly thereby
		"Deployment.SourceID.Location.Repo",
		"Deployment.SourceID.Location.Dir",
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

	defs := Defs{Clusters: Clusters{nick: &Cluster{BaseURL: "http://not"}}}

	d, err := BuildDeployment(defs, m, nick, sp, ih)

	if assert.NoError(err) {
		if assert.Len(d.DeployConfig.Volumes, 1) {
			assert.Equal("c", d.DeployConfig.Volumes[0].Container)
		}
		assert.Equal(nick, d.ClusterName)
	}
}

func TestDigest(t *testing.T) {
	tmpl := "got:%s expected:%s"
	expected := "3ea161adca77a01781628e8a7d24ad0e"
	d := &DeploymentID{
		ManifestID: ManifestID{
			Source: SourceLocation{
				Repo: "fake.tld/org/" + "project",
				Dir:  "down/here",
			},
		},
		Cluster: "test-cluster",
	}
	dStr := fmt.Sprintf("%x", d.Digest())
	if dStr != expected {
		t.Fatalf(tmpl, dStr, expected)
	} else {
		t.Logf(tmpl, dStr, expected)
	}
}
