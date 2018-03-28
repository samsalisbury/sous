//+build smoke

package smoke

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

const simpleServer = `
FROM alpine
CMD if [ -z "$T" ]; then T=2; fi; echo -n "Sleeping ${T}s..."; sleep $T; echo "Done"; echo "Listening on :$PORT0"; while true; do echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0; done`

func TestSousNewdeploy(t *testing.T) {
	f := setupEnv(t, "TestSousNewdeploy")

	// Setup project git repo.
	projectDir := makeGitRepo(t, f.BaseDir, "projects/project1", GitRepoSpec{
		UserName:  "Sous User 1",
		UserEmail: "sous-user1@example.com",
		OriginURL: "git@github.com:opentable/bogus/repo1",
	})
	makeFileString(t, projectDir, "Dockerfile", simpleServer)
	mustDoCMD(t, projectDir, "git", "add", "Dockerfile")
	mustDoCMD(t, projectDir, "git", "commit", "-m", "Add Dockerfile")

	sous := f.Client

	// Dump sous version & config.
	t.Logf("Sous version: %s", sous.MustRun(t, "version"))
	sous.MustRun("config")

	// cd into project dir
	sous.Dir = projectDir

	// sous init
	sous.MustRun(t, "init")

	// sous manifest get > manifest
	manifest := sous.MustRun(t, "manifest", "get")

	// edit manifest
	manifest = strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)

	// sous manifest set < manifest
	manifestSetCmd := sous.Cmd(t, "manifest", "set")
	manifestSetCmd.Stdin = ioutil.NopCloser(bytes.NewReader([]byte(manifest)))
	if out, err := manifestSetCmd.CombinedOutput(); err != nil {
		t.Fatalf("manifest set failed: %s; output:\n%s", err, out)
	}

	// sous build
	sous.MustRun(t, "build", "-tag", "1.2.3")

	// sous newdeploy
	clientDebug := os.Getenv("SOUS_CLIENT_DEBUG") == "true"
	args := []string{"-cluster", "cluster1", "-tag", "1.2.3"}
	if clientDebug {
		args = append([]string{"-d"}, args...)
	}

	sous.MustRun(t, append([]string{"newdeploy"}, args...))
}
