//+build smoke

package smoke

import (
	"bytes"
	"fmt"
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

	projectDir := makeGitRepo(t, f.BaseDir, "projects/project1", GitRepoSpec{
		UserName:  "Sous User 1",
		UserEmail: "sous-user1@example.com",
		OriginURL: "git@github.com:opentable/bogus/repo1",
	})

	makeFileString(t, projectDir, "Dockerfile", simpleServer)
	mustDoCMD(t, projectDir, "git", "add", "Dockerfile")
	mustDoCMD(t, projectDir, "git", "commit", "-am", "Add Dockerfile")

	sous := f.Client

	sous.MustRun(t, "version")

	// sous init
	sous.Dir = projectDir
	sous.MustRun(t, "init")

	// sous manifest get > manifest
	manifestGetCmd := sous.Cmd(t, "manifest", "get")
	manifestBytes, err := manifestGetCmd.Output()
	if err != nil {
		//output, err := manifestGetCmd.CombinedOutput()
		//		if err != nil {
		t.Fatalf("sous manifest get failed: %s; output:\n%s", err, string(manifestBytes))
		//}
		//t.Fatalf("sous manifest get weirdly failed then succeeded")
	}

	fmt.Println(string(manifestBytes))

	// edit manifest
	manifest := strings.Replace(string(manifestBytes), "NumInstances: 0", "NumInstances: 1", -1)

	fmt.Print(manifest)

	// sous manifest set < manifest
	manifestSetCmd := sous.Cmd(t, "manifest", "set")
	manifestSetCmd.Stdin = ioutil.NopCloser(bytes.NewReader([]byte(manifest)))

	if out, err := manifestSetCmd.CombinedOutput(); err != nil {
		t.Fatalf("manifest set failed: %s; output:\n%s", err, out)
	}

	if _, err := sous.Run(t, "build", "-tag", "1.2.3"); err != nil {
		t.Fatal(err)
	}

	clientDebug := os.Getenv("SOUS_CLIENT_DEBUG") == "true"
	commands := []string{}

	if clientDebug {
		commands = []string{"newdeploy", "-d", "-cluster", "cluster1", "-tag", "1.2.3"}
	} else {
		commands = []string{"newdeploy", "-cluster", "cluster1", "-tag", "1.2.3"}
	}

	if _, err := sous.Run(t, commands...); err != nil {
		t.Fatal(err)
	}

}
