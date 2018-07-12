//+build smoke

package smoke

import (
	"testing"

	"github.com/opentable/sous/util/filemap"
)

// Define some Dockerfiles for use in tests.
const (
	simpleServer = `
FROM alpine:3.7
CMD if [ -z "$T" ]; then T=2; fi; echo -n "Sleeping ${T}s..."; sleep $T; echo "Done"; echo "Listening on :$PORT0"; while true; do echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0; done
`
	sleeper = `
FROM alpine:3.7
CMD echo -n Sleeping for 10s...; sleep 10; echo Done
`
	failer = `
FROM alpine:3.7
CMD echo -n Failing in 10s...; sleep 10; echo Failed; exit 1
`
)

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
