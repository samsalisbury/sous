//+build smoke

package smoke

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opentable/sous/util/filemap"
)

type ProjectMaker func() filemap.FileMap

type ProjectList struct {
	GroupName                   string
	HTTPServer, Sleeper, Failer ProjectMaker
}

var projects = struct {
	SingleDockerfile ProjectList
	SplitBuild       ProjectList
}{
	SingleDockerfile: ProjectList{
		GroupName:  "dockerbuild",
		HTTPServer: func() filemap.FileMap { return singleDockerfile(httpServer) },
		Sleeper:    func() filemap.FileMap { return singleDockerfile(sleepT) },
		Failer:     func() filemap.FileMap { return singleDockerfile(failImmediately) },
	},
	SplitBuild: ProjectList{
		GroupName:  "splitbuild",
		HTTPServer: func() filemap.FileMap { return splitBuild(httpServer) },
		Sleeper:    func() filemap.FileMap { return splitBuild(sleepT) },
		Failer:     func() filemap.FileMap { return splitBuild(failImmediately) },
	},
}

// Program is a POSIX shell script program.
type Program string

// All these shell scripts require explicit command termination with ; since
// they may be inlined later.
const (
	sleepT = Program(`
		if [ -z "$T" ]; then T=2; fi;
		echo -n "Sleeping ${T}s...";
		sleep $T;
		echo "Awake";
		`)
	httpServer = Program(`
		echo Listening on :$PORT0;
		while true; do
		  echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0;
		done;
		`)
	failImmediately = Program(`
		echo Failing now;
		exit 1
		`)
	exitImmediately = Program(`
		echo Done;
		`)
)

func simpleServer() filemap.FileMap {
	return singleDockerfile(httpServer)
}

func sleeper() filemap.FileMap {
	return singleDockerfile(sleepT)
}

func failer() filemap.FileMap {
	return singleDockerfile(failImmediately)
}

func (p Program) FormatForDockerfile() string {
	return strings.Replace(string(p), "\n", "\\\n", -1)
}

func (p Program) FormatAsShellFile() string {
	return fmt.Sprintf("#!/usr/bin/env sh\n%s", p)
}

func singleDockerfile(p Program) filemap.FileMap {
	return filemap.FileMap{
		"Dockerfile": fmt.Sprintf(
			"FROM alpine:3.7\nCMD %s", p.FormatForDockerfile()),
	}
}

func simpleServerSplitContainer() filemap.FileMap {
	return filemap.FileMap{
		"Dockerfile": `
			FROM alpine:3.7
			ENV SOUS_RUN_IMAGE_SPEC=/image-spec.json
			COPY image-spec.json /
			RUN mkdir /server
			COPY server.sh /server/
			`,
		"image-spec.json": `
			{
			  "image": {
			    "type": "Docker",
				"from": "alpine:3.2"
			  },
			  "files": [
			    {
				  "source": {"dir": "/server"},
			      "dest": {"dir": "/"}
			    }
			  ],
			  "exec": ["/server/server.sh"]
			}
			`,
		"server.sh": `#!/usr/bin/env sh
			echo "Listening on :$PORT0"
			while true; do
			  echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0
			done
			`,
	}
}

func splitBuild(p Program) filemap.FileMap {
	return filemap.FileMap{
		"Dockerfile": `
			FROM alpine:3.7
			ENV SOUS_RUN_IMAGE_SPEC=/image-spec.json
			COPY image-spec.json /
			RUN mkdir /server
			COPY server.sh /server/
			`,
		"image-spec.json": `
			{
			  "image": {
			    "type": "Docker",
				"from": "alpine:3.2"
			  },
			  "files": [
			    {
				  "source": {"dir": "/server"},
			      "dest": {"dir": "/"}
			    }
			  ],
			  "exec": ["/server/server.sh"]
			}
			`,
		"server.sh": p.FormatAsShellFile(),
	}
}

// setupProject creates a brand new git repo containing the provided Dockerfile,
// commits that Dockerfile, runs 'sous version' and 'sous config', and returns a
// sous TestClient in the project directory.
func setupProjectSingleDockerfile(t *testing.T, f *TestFixture, dockerfile string) *TestClient {
	return setupProject(t, f, filemap.FileMap{"Dockerfile": dockerfile})
}

func setupProject(t *testing.T, f *TestFixture, fm filemap.FileMap) *TestClient {
	t.Helper()
	// Setup project git repo.
	projectDir := makeGitRepo(t, f.Client.BaseDir, "projects/project1", GitRepoSpec{
		UserName:  "Sous User 1",
		UserEmail: "sous-user1@example.com",
		OriginURL: "git@github.com:user1/repo1.git",
	})
	if err := fm.Write(projectDir); err != nil {
		t.Fatalf("filemap.Write: %s", err)
	}
	for filePath := range fm {
		mustDoCMD(t, projectDir, "git", "add", filePath)
	}
	mustDoCMD(t, projectDir, "git", "commit", "-m", "Initial Commit")

	client := f.Client

	// cd into project dir
	client.Dir = projectDir

	// Dump sous version & config.
	t.Logf("Sous version: %s", client.MustRun(t, "version", nil))
	client.MustRun(t, "config", nil)

	return client
}
