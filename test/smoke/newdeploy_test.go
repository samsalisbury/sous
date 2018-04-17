//+build smoke

package smoke

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

const simpleServer = `
FROM alpine
CMD if [ -z "$T" ]; then T=2; fi; echo -n "Sleeping ${T}s..."; sleep $T; echo "Done"; echo "Listening on :$PORT0"; while true; do echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0; done`

// setupProject creates a brand new git repo containing the provided Dockerfile,
// runs sous init, then manifest get/set to bump instances to 1 in all clusters.
func setupProject(t *testing.T, f Fixture, dockerfile string) TestClient {
	t.Helper()
	// Setup project git repo.
	projectDir := makeGitRepo(t, f.BaseDir, "projects/project1", GitRepoSpec{
		UserName:  "Sous User 1",
		UserEmail: "sous-user1@example.com",
		OriginURL: "git@github.com:opentable/bogus/repo1",
	})
	makeFileString(t, projectDir, "Dockerfile", dockerfile)
	mustDoCMD(t, projectDir, "git", "add", "Dockerfile")
	mustDoCMD(t, projectDir, "git", "commit", "-m", "Add Dockerfile")

	sous := f.Client

	// Dump sous version & config.
	t.Logf("Sous version: %s", sous.MustRun(t, "version"))
	sous.MustRun(t, "config")

	// cd into project dir
	sous.Dir = projectDir

	return sous
}

func initProjectNoFlavor(t *testing.T, sous TestClient) {
	t.Helper()
	// Prepare manifest.
	sous.MustRun(t, "init")
	manifest := sous.MustRun(t, "manifest", "get")
	manifest = strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)
	manifestSetCmd := sous.Cmd(t, "manifest", "set")
	manifestSetCmd.Stdin = ioutil.NopCloser(bytes.NewReader([]byte(manifest)))
	if out, err := manifestSetCmd.CombinedOutput(); err != nil {
		t.Fatalf("manifest set failed: %s; output:\n%s", err, out)
	}
}

func initProjectWithFlavor(t *testing.T, sous TestClient, flavor string) {
	t.Helper()
	// Prepare manifest.
	sous.MustRun(t, "init", "-flavor", flavor)
	manifest := sous.MustRun(t, "manifest", "get", "-flavor", flavor)
	manifest = strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)
	manifestSetCmd := sous.Cmd(t, "manifest", "set", "-flavor", flavor)
	manifestSetCmd.Stdin = ioutil.NopCloser(bytes.NewReader([]byte(manifest)))
	if out, err := manifestSetCmd.CombinedOutput(); err != nil {
		t.Fatalf("manifest set failed: %s; output:\n%s", err, out)
	}
}

func TestSousNewdeploy(t *testing.T) {

	t.Run("simple", func(t *testing.T) {
		f := setupEnv(t)
		sous := setupProject(t, f, simpleServer)
		initProjectNoFlavor(t, sous)
		// Build and deploy.
		sous.MustRun(t, "build", "-tag", "1.2.3")
		sous.MustRun(t, "newdeploy", "-cluster", "cluster1", "-tag", "1.2.3")
	})

	t.Run("flavors", func(t *testing.T) {
		f := setupEnv(t)
		sous := setupProject(t, f, simpleServer)
		flavor := "flavor1"
		initProjectWithFlavor(t, sous, flavor)
		sous.MustRun(t, "build", "-tag", "1.2.3")
		sous.MustRun(t, "newdeploy", "-cluster", "cluster1", "-tag", "1.2.3", "-flavor", flavor)
	})

}
