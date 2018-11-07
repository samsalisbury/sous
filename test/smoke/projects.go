package smoke

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opentable/sous/ext/git"
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

type sousProject struct {
	// sousClient is a sous client at the root of the repo.
	*sousClient
	// git is a git client at the root of the repo.
	git *gitClient
	// files contains the project files.
	files filemap.FileMap
	// repo is the isolated repo name.
	repo string
}

type sousProjectConfig struct {
	gitRepoSpec *gitRepoSpec
}

// setupProject returns a *sousProject with an isolated repo name (isolated
// in terms of the name of the test t.
func setupProject(t *testing.T, f *fixture, fm filemap.FileMap, config ...func(*sousProjectConfig)) *sousProject {
	t.Helper()

	// Setup project git repo.
	g := newGitClient(t, f.fixtureConfig, "gitclient1")

	projectDir := f.newEmptyDir("project1")
	g.CD(projectDir)

	// TODO SS: This ToLower call will not be necessary once we properly handle
	// repos and offsets that contain upper-case letters. Remove ToLower call
	// once that is done.
	isolatedRepoName := strings.ToLower(t.Name())

	origin := "git@github.com:" + isolatedRepoName + ".git"

	origin = strings.ToLower(origin)
	c := &sousProjectConfig{
		gitRepoSpec: &gitRepoSpec{
			UserName:  "Sous User 1",
			UserEmail: "sous-user1@example.com",
			OriginURL: origin,
		},
	}
	for _, f := range config {
		f(c)
	}

	repoName, err := git.CanonicalRepoURL(origin)
	if err != nil {
		t.Fatalf("Setup failed to generate valid git origin URL: %s", err)
	}

	g.init(t, f.fixtureConfig, *c.gitRepoSpec)

	if err := fm.Write(projectDir); err != nil {
		t.Fatalf("filemap.Write: %s", err)
	}
	for filePath := range fm {
		g.MustRun(t, "add", nil, filePath)
	}

	g.MustRun(t, "commit", nil, "-m", "initial commit: "+f.TestName)

	client := f.Client
	client.Dir = projectDir

	// Dump sous version & config.
	if !quiet() {
		client.MustRun(t, "version", nil)
	}

	return &sousProject{
		sousClient: client,
		git:        g,
		files:      fm,
		repo:       repoName,
	}
}
