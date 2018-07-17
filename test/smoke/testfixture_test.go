//+build smoke

package smoke

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
)

type TestFixture struct {
	EnvDesc     desc.EnvDesc
	Cluster     TestBunchOfSousServers
	Client      *TestClient
	BaseDir     string
	Singularity *Singularity
	// ClusterSuffix is used to add a suffix to each generated cluster name.
	// This can be used to segregate parallel tests.
	ClusterSuffix string
	Parent        *ParallelTestFixture
	TestName      string
	UserEmail     string
	Projects      ProjectList
	knownToFail   bool
}

var sousBin = mustGetSousBin()

func newTestFixture(t *testing.T, envDesc desc.EnvDesc, parent *ParallelTestFixture, nextAddr func() string, fcfg fixtureConfig) *TestFixture {
	t.Helper()
	t.Parallel()
	if testing.Short() {
		t.Skipf("-short flag present")
	}
	baseDir := getDataDir(t)

	clusterSuffix := strings.Replace(t.Name(), "/", "_", -1)
	fmt.Fprintf(os.Stdout, "Cluster suffix: %s", clusterSuffix)

	singularity := NewSingularity(envDesc.SingularityURL())
	singularity.ClusterSuffix = clusterSuffix

	state := sous.StateFixture(sous.StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
		ClusterSuffix: clusterSuffix,
	})

	addURLsToState(state, envDesc)

	fcfg.startState = state

	c, err := newBunchOfSousServers(t, baseDir, nextAddr, fcfg)
	if err != nil {
		t.Fatalf("setting up test cluster: %s", err)
	}

	if err := c.Configure(t, envDesc, fcfg); err != nil {
		t.Fatalf("configuring test cluster: %s", err)
	}

	if err := c.Start(t, sousBin); err != nil {
		t.Fatalf("starting test cluster: %s", err)
	}

	primaryServer := "http://" + c.Instances[0].Addr
	userEmail := "sous_client1@example.com"

	tf := &TestFixture{
		Cluster:       *c,
		BaseDir:       baseDir,
		Singularity:   singularity,
		ClusterSuffix: clusterSuffix,
		Parent:        parent,
		TestName:      t.Name(),
		UserEmail:     userEmail,
		Projects:      fcfg.projects,
	}
	client := makeClient(tf, baseDir, sousBin)
	if err := client.Configure(primaryServer, envDesc.RegistryName(), userEmail); err != nil {
		t.Fatal(err)
	}
	tf.Client = client
	return tf
}

func (f *TestFixture) Teardown(t *testing.T) {
	t.Helper()
	if shouldStopServers(t) {
		if err := f.Cluster.Stop(); err != nil {
			t.Errorf("failed to stop cluster: %s", err)
		}
	}
	if shouldCleanFiles(t) {
		f.Clean(t)
	}
}

func shouldStopServers(t *testing.T) bool {
	// TODO SS: Make this configurable.
	return !t.Failed()
}

func shouldCleanFiles(t *testing.T) bool {
	// TODO SS: Make this configurable.
	return !t.Failed()
}

func (f *TestFixture) Clean(t *testing.T) {
	t.Helper()
	contents, err := ioutil.ReadDir(f.BaseDir)
	if err != nil {
		t.Errorf("failed to clean up: read dir: %s", err)
		return
	}
	for _, file := range contents {
		filePath := filepath.Join(f.BaseDir, file.Name())
		if err := os.RemoveAll(filePath); err != nil {
			t.Errorf("failed to clean up: deleting %s: %s", file, err)
		}
		fileName := "FAILED"
		if !t.Failed() {
			fileName = "PASSED"
		}
		passFailPath := filepath.Join(f.BaseDir, fileName)
		if err := ioutil.WriteFile(passFailPath, nil, os.ModePerm); err != nil {
			t.Errorf("cleaned up but failed to to write passFailPath: %s", err)
		}
	}
}

// DefaultSingReqID returns the default singularity request ID for the\
// DeploymentID derived from the passed flags. If flags do not have both
// repo and cluster set this task is impossible and thus fails the test
// immediately.
func (f *TestFixture) DefaultSingReqID(t *testing.T, flags *sousFlags) string {
	t.Helper()
	if flags.repo == "" {
		t.Fatalf("flags.repo empty")
	}
	if flags.cluster == "" {
		t.Fatalf("flags.cluster empty")
	}
	did := sous.DeploymentID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: flags.repo,
				Dir:  flags.offset,
			},
			Flavor: flags.flavor,
		},
		Cluster: flags.cluster,
	}
	return f.Singularity.DefaultReqID(t, did)
}

// IsolatedClusterName returns a cluster name unique to this test fixture.
func (f *TestFixture) IsolatedClusterName(baseName string) string {
	return baseName + f.ClusterSuffix
}

// IsolatedVersionTag returns an all-lowercase unique version tag (unique per
// test-run, subsequent runs will use the same tag). These version tags should
// be compatible natively as both Sous and Docker tags.
func (f *TestFixture) IsolatedVersionTag(t *testing.T, baseTag string) string {
	t.Helper()
	v, err := semv.Parse(baseTag)
	if err != nil {
		t.Fatalf("version tag %q not semver: %s", baseTag, err)
	}
	if v.Meta != "" {
		t.Fatalf("version tag %q contains metatdata field", baseTag)
	}
	suffix := strings.Replace(f.ClusterSuffix, "_", "-", -1)
	if v.Pre != "" {
		return strings.ToLower(baseTag + suffix)
	}
	return strings.ToLower(baseTag + "-" + suffix)
}

func (f *TestFixture) ReportStatus(t *testing.T) {
	t.Helper()
	f.Parent.recordTestStatus(t)
}

func (f *TestFixture) KnownToFailHere(t *testing.T) {
	t.Helper()
	const skipKnownFailuresEnvVar = "EXCLUDE_KNOWN_FAILING_TESTS"
	if os.Getenv(skipKnownFailuresEnvVar) == "YES" {
		f.knownToFail = true
		t.Skipf("This test is known to fail and you set %s=YES",
			skipKnownFailuresEnvVar)
	}
}
