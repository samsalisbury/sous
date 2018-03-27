//+build smoke

package smoke

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

func TestSousNewdeploy(t *testing.T) {
	f := setupEnv(t, "TestSousNewdeploy")

	dockerfile := `FROM alpine
CMD if [ -z "$T" ]; then T=2; fi; echo -n "Sleeping ${T}s..."; sleep $T; echo "Done"; echo "Listening on :$PORT0"; while true; do echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0; done
`

	projectDir := path.Join(os.TempDir(), "project1")
	if err := os.RemoveAll(projectDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectDir, 0777); err != nil {
		t.Fatal(err)
	}
	dockerfilePath := path.Join(projectDir, "Dockerfile")
	if err := ioutil.WriteFile(dockerfilePath, []byte(dockerfile), 0777); err != nil {
		t.Fatal(err)
	}

	if err := func() error {
		gdmDir := projectDir
		if err := doCMD(gdmDir, "git", "init"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "config", "user.name", "Sous User"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "config", "user.email", "sous-user@example.com"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "add", "Dockerfile"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "commit", "-am", "add Dockerfile"); err != nil {
			return err
		}
		if err := doCMD(gdmDir, "git", "remote", "add", "origin", "git@github.com:opentable/bogus/repo.git"); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		t.Fatal(err)
	}

	sous := f.Client

	if _, err := sous.Run(t, "version"); err != nil {
		t.Fatal(err)
	}

	// sous init
	sous.Dir = projectDir
	if _, err := sous.Run(t, "init"); err != nil {
		t.Fatal(err)
	}

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
