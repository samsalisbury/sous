package smoke

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opentable/sous/util/filemap"
)

type projectMaker func() filemap.FileMap

type projectList struct {
	GroupName                   string
	HTTPServer, Sleeper, Failer projectMaker
}

var projects = struct {
	SingleDockerfile projectList
	SplitBuild       projectList
}{
	SingleDockerfile: projectList{
		GroupName:  "dockerbuild",
		HTTPServer: func() filemap.FileMap { return singleDockerfile(httpServer) },
		Sleeper:    func() filemap.FileMap { return singleDockerfile(sleepT) },
		Failer:     func() filemap.FileMap { return singleDockerfile(failImmediately) },
	},
	SplitBuild: projectList{
		GroupName:  "splitbuild",
		HTTPServer: func() filemap.FileMap { return splitBuild(httpServer) },
		Sleeper:    func() filemap.FileMap { return splitBuild(sleepT) },
		Failer:     func() filemap.FileMap { return splitBuild(failImmediately) },
	},
}

// program is a POSIX shell script program.
type program string

// All these shell scripts require explicit command termination with ; since
// they may be inlined later.
const (
	sleepT = program(`
		if [ -z "$T" ]; then T=2; fi;
		echo -n "Sleeping ${T}s...";
		sleep $T;
		echo "Awake";
		`)
	httpServer = program(`
		echo Listening on :$PORT0;
		while true; do
		  echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0;
		done;
		`)
	failImmediately = program(`
		echo Failing now;
		exit 1
		`)
	exitImmediately = program(`
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

// String returns p as a string with spaces trimmed.
func (p program) String() string {
	return strings.TrimSpace(string(p))
}

func (p program) FormatForDockerfile() string {
	return strings.Replace(p.String(), "\n", " \\\n", -1)
}

func (p program) FormatAsShellFile() string {
	return fmt.Sprintf("#!/usr/bin/env sh\n\n%s", p)
}

func singleDockerfile(p program) filemap.FileMap {
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

func splitBuild(p program) filemap.FileMap {
	return filemap.FileMap{
		"Dockerfile": `
			FROM alpine:3.7
			ENV SOUS_RUN_IMAGE_SPEC=/image-spec.json
			COPY image-spec.json /
			RUN mkdir /server
			COPY server.sh /server/
			CMD echo This is a builder image.
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

func setupProject(t *testing.T, f *fixture, fm filemap.FileMap) *sousClient {
	t.Helper()

	// Setup project git repo.
	g := newGitClient(t, f, f.Client.BaseDir)

	projectDir := g.configureRepo(t, "projects/project1", gitRepoSpec{
		UserName:  "Sous User 1",
		UserEmail: "sous-user1@example.com",
		OriginURL: "git@github.com:user1/repo1.git",
	})

	if err := fm.Write(projectDir); err != nil {
		t.Fatalf("filemap.Write: %s", err)
	}
	for filePath := range fm {
		g.MustRun(t, "add", nil, filePath)
	}

	g.MustRun(t, "commit", nil, "-m", "initial commit")

	client := f.Client
	client.Dir = projectDir

	// Dump sous version & config.
	if !quiet() {
		client.MustRun(t, "version", nil)
	}

	return client
}
