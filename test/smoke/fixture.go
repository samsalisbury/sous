package smoke

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/testmatrix"
	"github.com/samsalisbury/semv"
)

// fixtureBase is generic fixture stuff.
type fixtureBase struct {
	TestName string
	BaseDir  string
	Finished chan struct{}

	knownToFail bool
}

// fixtureConfig is a priori sous-specific fixture stuff.
type fixtureConfig struct {
	fixtureBase
	Scenario scenario
	// ClusterSuffix is used to add a suffix to each generated cluster name.
	// This can be used to segregate parallel tests.
	ClusterSuffix string
	EnvDesc       desc.EnvDesc
	UserEmail     string
	Projects      projectList
	InitialState  *sous.State

	Singularity *testSingularity
}

// fixture is the full rich fixture object passed to tests.
type fixture struct {
	fixtureConfig
	Cluster bunchOfSousServers
	Client  *sousClient
}

var sousBin = mustGetSousBin()

func newFixtureConfig(testName string, s testmatrix.Scenario) fixtureConfig {
	base := fixtureBase{
		TestName: testName,
		BaseDir:  getDataDir(testName),
		Finished: make(chan struct{}),
	}

	scenario := unwrapScenario(s)
	envDesc := getEnvDesc()
	clusterSuffix := strings.Replace(testName, "/", "_", -1)
	s9y := newSingularity(envDesc.SingularityURL())
	s9y.ClusterSuffix = clusterSuffix
	state := sous.StateFixture(sous.StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
		ClusterSuffix: clusterSuffix,
	})
	addURLsToState(state, envDesc)
	return fixtureConfig{
		fixtureBase:   base,
		Scenario:      scenario,
		ClusterSuffix: clusterSuffix,
		EnvDesc:       getEnvDesc(),
		UserEmail:     "sous_client1@example.com",
		Projects:      scenario.projects,
		InitialState:  state,
		Singularity:   s9y,
	}
}

func (f *fixtureBase) absPath(path string) string {
	if strings.HasPrefix(path, f.BaseDir) {
		return path
	}
	return filepath.Join(f.BaseDir, path)
}

func (f *fixtureBase) newEmptyDir(path string) string {
	path = f.absPath(path)
	makeEmptyDirAbs(path)
	return path
}

func (f *fixtureBase) newBin(t *testing.T, path, instanceName string) Bin {
	binBaseDir := f.absPath(filepath.Join("actors", instanceName))
	return NewBin(t, path, instanceName, binBaseDir, f.BaseDir, f.Finished)
}

func newConfiguredFixture(t *testing.T, s testmatrix.Scenario, mod ...func(*fixtureConfig)) *fixture {
	config := newFixtureConfig(t.Name(), s)

	for _, m := range mod {
		m(&config)
	}

	boss, err := newBunchOfSousServers(t, config)
	if err != nil {
		t.Fatalf("setting up test cluster: %s", err)
	}

	if err := boss.configure(t, config); err != nil {
		t.Fatalf("configuring test cluster: %s", err)
	}

	boss.Start(t)

	primaryServer := "http://" + boss.Instances[0].Addr

	tf := &fixture{
		fixtureConfig: config,
		Cluster:       *boss,
	}
	client := makeClient(t, config, sousBin)
	if err := client.Configure(primaryServer, config.EnvDesc.RegistryName(), config.UserEmail); err != nil {
		t.Fatal(err)
	}
	tf.Client = client
	return tf
}

// newFixture transforms a testmatrix.Scenario into a sous-specific fixture.
func newFixture(t *testing.T, s testmatrix.Scenario) testmatrix.Fixture {
	return newConfiguredFixture(t, s)
}

// Teardown performs conditional cleanup of resources used in the test.
// This includes stopping servers and deleting intermediate test data (config
// files, git repos, logs etc.) in the case that the test passed.
func (f *fixture) Teardown(t *testing.T) {
	t.Helper()
	close(f.Finished)
	if shouldStopServers(t) {
		time.Sleep(time.Second) // TODO: Fix synchronisation.
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
	if sup.TestCount() == 1 {
		return false // When running a single test do not clean up.
	}
	return !t.Failed()
}

func (f *fixture) Clean(t *testing.T) {
	t.Helper()
	contents, err := ioutil.ReadDir(f.BaseDir)
	if err != nil {
		t.Errorf("failed to clean up: read dir: %s", err)
		return
	}
	for _, file := range contents {
		filePath := filepath.Join(f.BaseDir, file.Name())
		if err := os.RemoveAll(filePath); err != nil {
			t.Errorf("failed to clean up: deleting %s: %s", filePath, err)
		}
		fileName := "FAILED"
		if !t.Failed() {
			fileName = "PASSED"
		}
		passFailPath := filepath.Join(f.BaseDir, fileName)
		if err := ioutil.WriteFile(passFailPath, nil, os.ModePerm); err != nil {
			t.Errorf("cleaned up but failed to to write %s: %s", passFailPath, err)
		}
	}
}

// DefaultSingReqID returns the default singularity request ID for the
// DeploymentID derived from the passed flags. If flags do not have both
// repo and cluster set this task is impossible and thus fails the test
// immediately.
func (f *fixture) DefaultSingReqID(t *testing.T, flags *sousFlags) string {
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

func ensureSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		return s
	}
	return s + suffix
}

// IsolatedClusterName returns a cluster name unique to this test fixture.
func (f *fixtureConfig) IsolatedClusterName(baseName string) string {
	return ensureSuffix(baseName, f.ClusterSuffix)
}

func (f *fixtureConfig) IsolatedRequestID(baseName string) string {
	return ensureSuffix(baseName, f.ClusterSuffix)
}

// IsolatedVersionTag returns an all-lowercase unique version tag (unique per
// test-run, subsequent runs will use the same tag). These version tags are
// compatible natively as both Sous and Docker tags for convenience.
func (f *fixtureConfig) IsolatedVersionTag(baseTag string) string {
	v, err := semv.Parse(baseTag)
	if err != nil {
		panic(fmt.Errorf("version tag %q not semver: %s", baseTag, err))
	}
	if v.Meta != "" {
		panic(fmt.Errorf("version tag %q contains metatdata field", baseTag))
	}
	suffix := strings.Replace(f.ClusterSuffix, "_", "-", -1)
	if strings.HasSuffix(baseTag, suffix) {
		return baseTag
	}
	if v.Pre != "" {
		return strings.ToLower(baseTag + suffix)
	}
	return strings.ToLower(baseTag + "-" + suffix)
}

// KnownToFailHere cauuses the test to be skipped from this point on if
// the environment variable EXCLUDE_KNOWN_FAILING_TESTS=YES.
func (f *fixture) KnownToFailHere(t *testing.T) {
	t.Helper()
	const skipKnownFailuresEnvVar = "EXCLUDE_KNOWN_FAILING_TESTS"
	if os.Getenv(skipKnownFailuresEnvVar) == "YES" {
		f.knownToFail = true
		t.Skipf("This test is known to fail and you set %s=YES",
			skipKnownFailuresEnvVar)
	}
}
